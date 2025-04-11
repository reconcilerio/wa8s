/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"reconciler.io/runtime/reconcilers"
	rtime "reconciler.io/runtime/time"
	"reconciler.io/runtime/validation"
	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	"reconciler.io/wa8s/components"
	"reconciler.io/wa8s/registry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ConfigStoreStasher             = reconcilers.NewStasher[map[string]string](reconcilers.StashKey("wa8s.reconciler.io/config-store"))
	ComponentStasher               = reconcilers.NewStasher[[]byte](reconcilers.StashKey("wa8s.reconciler.io/component"))
	ComponentConfigStasher         = reconcilers.NewStasher[registry.WasmConfigFile](reconcilers.StashKey("wa8s.reconciler.io/component-config"))
	ComponentTraceStasher          = reconcilers.NewStasher[[]componentsv1alpha1.ComponentSpan](reconcilers.StashKey("wa8s.reconciler.io/component-trace"))
	CompositionDependenciesStasher = reconcilers.NewStasher[[]components.ResolvedComponent](reconcilers.StashKey("wa8s.reconciler.io/composition-dependencies"))
	RepositoryKeychainStasher      = reconcilers.NewStasher[authn.Keychain](reconcilers.StashKey("wa8s.reconciler.io/repository-keychain"))
	RepositoryDigestStasher        = reconcilers.NewStasher[name.Digest](reconcilers.StashKey("wa8s.reconciler.io/repository-digest"))
	RepositoryTagStasher           = reconcilers.NewStasher[name.Tag](reconcilers.StashKey("wa8s.reconciler.io/repository-tag"))
)

var (
	// ErrTransient captures an error that is of the moment, retrying the request may succeed. Meaningful state about the error has been captured on the status
	ErrTransient = errors.Join(reconcilers.ErrQuiet, reconcilers.ErrHaltSubReconcilers)
	// ErrDurable is permanent given the current state, the request should not be retried until the observed state has changed. Meaningful state about the error has been captured on the status
	ErrDurable = errors.Join(reconcilers.ErrQuiet, reconcilers.ErrHaltSubReconcilers)
	// ErrGenerationMismatch a referenced resource's .metadata.generation and .status.observedGeneration are out of sync. Treat as a transient error as this state is expected and we should avoid flapping
	ErrGenerationMismatch = errors.Join(ErrTransient, reconcilers.ErrSkipStatusUpdate)
)

type DebounceTransientErrors[Type client.Object, ListType client.ObjectList] struct {
	// Name used to identify this reconciler.  Defaults to `ForEach`. Ideally unique, but
	// not required to be so.
	//
	// +optional
	Name string

	// ListType is the listing type for the type. For example, PodList is the list type for Pod.
	// Required when the generic type is not a struct, or is unstructured.
	//
	// +optional
	ListType ListType

	// Setup performs initialization on the manager and builder this reconciler
	// will run with. It's common to setup field indexes and watch resources.
	//
	// +optional
	Setup func(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error

	// TransientErrorThreshold is the number of ErrTransient reconciles encountered for a resource
	// after which the returned error is ErrDurable
	TransientErrorThreshold uint8

	// Reconciler to be called for each iterable item
	Reconciler reconcilers.SubReconciler[Type]

	lazyInit     sync.Once
	m            sync.Mutex
	lastPurge    time.Time
	errorCounter map[types.UID]debounceTransientErrorCounter
}

func (r *DebounceTransientErrors[T, LT]) SetupWithManager(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error {
	r.init()

	log := logr.FromContextOrDiscard(ctx).
		WithName(r.Name)
	ctx = logr.NewContext(ctx, log)

	if err := r.Validate(ctx); err != nil {
		return err
	}
	if err := r.Reconciler.SetupWithManager(ctx, mgr, bldr); err != nil {
		return err
	}
	if r.Setup == nil {
		return nil
	}
	return r.Setup(ctx, mgr, bldr)
}

func (r *DebounceTransientErrors[T, LT]) init() {
	r.lazyInit.Do(func() {
		if r.Name == "" {
			r.Name = "DebounceTransientErrors"
		}
		if isNil(r.ListType) {
			var nilLT LT
			r.ListType = newEmpty(nilLT).(LT)
		}
		if r.TransientErrorThreshold == 0 {
			r.TransientErrorThreshold = 3
		}
		r.errorCounter = map[types.UID]debounceTransientErrorCounter{}
	})
}

func (r *DebounceTransientErrors[T, LT]) checkStaleCounters(ctx context.Context) {
	now := rtime.RetrieveNow(ctx)
	log := logr.FromContextOrDiscard(ctx)

	r.m.Lock()
	defer r.m.Unlock()

	if r.lastPurge.IsZero() {
		r.lastPurge = now
		return
	}
	if r.lastPurge.Add(24 * time.Hour).After(now) {
		return
	}

	log.Info("purging stale resource counters")

	c := reconcilers.RetrieveConfigOrDie(ctx)
	list := r.ListType.DeepCopyObject().(LT)
	if err := c.List(ctx, list); err != nil {
		log.Error(err, "purge failed to list resources")
		return
	}

	validIds := sets.New[types.UID]()
	for _, item := range extractItems[T](list) {
		validIds.Insert(item.GetUID())
	}

	counterIds := sets.New[types.UID]()
	for uid := range r.errorCounter {
		counterIds.Insert(uid)
	}

	for _, uid := range counterIds.Difference(validIds).UnsortedList() {
		log.V(2).Info("purging counter", "id", uid)
		delete(r.errorCounter, uid)
	}

	r.lastPurge = now
}

func (r *DebounceTransientErrors[T, LT]) Validate(ctx context.Context) error {
	r.init()

	// validate Reconciler
	if r.Reconciler == nil {
		return fmt.Errorf("DebounceTransientErrors %q must implement Reconciler", r.Name)
	}
	if validation.IsRecursive(ctx) {
		if v, ok := r.Reconciler.(validation.Validator); ok {
			if err := v.Validate(ctx); err != nil {
				return fmt.Errorf("DebounceTransientErrors %q must have a valid Reconciler: %w", r.Name, err)
			}
		}
	}

	return nil
}

func (r *DebounceTransientErrors[T, LT]) Reconcile(ctx context.Context, resource T) (reconcilers.Result, error) {
	log := logr.FromContextOrDiscard(ctx).
		WithName(r.Name)
	ctx = logr.NewContext(ctx, log)

	defer r.checkStaleCounters(ctx)

	result, err := r.Reconciler.Reconcile(ctx, resource)

	if err == nil || errors.Is(err, ErrDurable) {
		delete(r.errorCounter, resource.GetUID())
		return result, err
	}

	// concurrent map access is ok, since keys are resources specific and a given resource will never be processed concurrently
	counter, ok := r.errorCounter[resource.GetUID()]
	if !ok || counter.ResourceVersion != resource.GetResourceVersion() {
		counter = debounceTransientErrorCounter{
			ResourceVersion: resource.GetResourceVersion(),
			Count:           0,
		}
	}

	// check overflow before incrementing
	if counter.Count != uint8(255) {
		counter.Count = counter.Count + 1
		r.errorCounter[resource.GetUID()] = counter
	}

	if counter.Count < r.TransientErrorThreshold {
		// suppress status update
		return result, errors.Join(err, reconcilers.ErrSkipStatusUpdate, reconcilers.ErrQuiet)
	}

	return result, err
}

type debounceTransientErrorCounter struct {
	ResourceVersion string
	Count           uint8
}

// isNil returns true if the value is nil, false if the value is not nilable or not nil
func isNil(val interface{}) bool {
	if val == nil {
		return true
	}
	if !isNilable(val) {
		return false
	}
	return reflect.ValueOf(val).IsNil()
}

// isNilable returns true if the value can be nil
func isNilable(val interface{}) bool {
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Chan:
		return true
	case reflect.Func:
		return true
	case reflect.Interface:
		return true
	case reflect.Map:
		return true
	case reflect.Ptr:
		return true
	case reflect.Slice:
		return true
	default:
		return false
	}
}

// newEmpty returns a new empty value of the same underlying type, preserving the existing value
func newEmpty(x interface{}) interface{} {
	t := reflect.TypeOf(x).Elem()
	return reflect.New(t).Interface()
}

// extractItems returns a typed slice of objects from an object list
func extractItems[T client.Object](list client.ObjectList) []T {
	items := []T{}
	listValue := reflect.ValueOf(list).Elem()
	itemsValue := listValue.FieldByName("Items")
	for i := 0; i < itemsValue.Len(); i++ {
		itemValue := itemsValue.Index(i)
		var item T
		switch itemValue.Kind() {
		case reflect.Pointer:
			item = itemValue.Interface().(T)
		case reflect.Interface:
			item = itemValue.Interface().(T)
		case reflect.Struct:
			item = itemValue.Addr().Interface().(T)
		default:
			panic(fmt.Errorf("unknown type %s for Items slice, expected Pointer or Struct", itemValue.Kind().String()))
		}
		items = append(items, item)
	}
	return items
}
