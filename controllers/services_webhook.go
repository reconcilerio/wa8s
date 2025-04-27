/*
Copyright 2025.

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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
)

var ErrUnknownSecret = errors.New("unknown secret")

var ServiceCredentialFinalizer = fmt.Sprintf("%s/finalizer", servicesv1alpha1.GroupVersion.Group)

func ServicesWebhook(mgr ctrl.Manager, c reconcilers.Config, addr string) manager.Runnable {
	return &servicesWebhooks{
		Addr:   addr,
		Config: c,
		mgr:    mgr,
	}
	// log := mgr.GetLogger().WithName("ServicesWebhook")
	// return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	log.Info("request", "method", r.Method, "path", r.RequestURI)
	// 	w.WriteHeader(204)
	// })
}

type servicesWebhooks struct {
	Addr   string
	Config reconcilers.Config
	mgr    ctrl.Manager
}

func (s *servicesWebhooks) Start(ctx context.Context) error {
	ctx = reconcilers.StashConfig(ctx, s.Config)
	ctx = reconcilers.StashOriginalConfig(ctx, s.Config)
	log, err := logr.FromContext(ctx)
	if err != nil {
		log = s.mgr.GetLogger()
	}
	log = log.WithName("ServicesWebhook")
	ctx = logr.NewContext(ctx, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/services/credentials/fetch", s.fetchCredentials(ctx))
	mux.HandleFunc("/services/credentials/publish", s.publishCredentials(ctx))
	mux.HandleFunc("/services/credentials/destroy", s.destroyCredentials(ctx))

	return http.ListenAndServe(s.Addr, mux)
}

func (s *servicesWebhooks) lookupInstanceID(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("lookupInstanceID")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI)

		bindingId := r.Header.Get("service-binding-id")
		if bindingId == "" {
			log.Error(fmt.Errorf("missing service-binding-id header"), "missing service-binding-id header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		secret, err := s.loadSecret(ctx, bindingId)
		if err != nil {
			log.Error(err, "unable to load binding")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if secret == nil {
			log.Info("unknown binding")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Add("service-instance-id", secret.Labels["services.wa8s.reconciler.io/instance-id"])
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *servicesWebhooks) fetchCredentials(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("fetchCredentials")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI)

		bindingId := r.Header.Get("service-binding-id")
		if bindingId == "" {
			log.Error(fmt.Errorf("missing service-binding-id header"), "missing service-binding-id header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		creds, err := s.retrieveCredentials(ctx, bindingId)
		if err != nil {
			log.Error(err, "unable to load creds")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(creds)
	}
}

func (s *servicesWebhooks) publishCredentials(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("publishCredentials")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI)

		bindingId := r.Header.Get("service-binding-id")
		if bindingId == "" {
			log.Error(fmt.Errorf("missing service-binding-id header"), "missing service-binding-id header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var creds map[string]string
		if err := json.Unmarshal([]byte(r.Header.Get("service-credentials")), &creds); err != nil {
			log.Error(err, "unable to decode service credentials")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := s.updateCredentials(ctx, bindingId, creds)
		if err != nil {
			log.Error(err, "failed to update service-binding-id")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *servicesWebhooks) destroyCredentials(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("destroyCredentials")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI)

		bindingId := r.Header.Get("service-binding-id")
		if bindingId == "" {
			log.Error(fmt.Errorf("missing service-binding-id header"), "missing service-binding-id header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := s.deleteCredentials(ctx, bindingId)
		if err != nil && !errors.Is(err, ErrUnknownSecret) {
			log.Error(err, "failed to destroy service-binding-id")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *servicesWebhooks) retrieveCredentials(ctx context.Context, id string) (map[string]string, error) {
	secret, err := s.loadSecret(ctx, id)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, ErrUnknownSecret
	}

	credentials := map[string]string{}
	for key, value := range secret.Data {
		credentials[key] = string(value)
	}

	return credentials, nil
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=update

func (s *servicesWebhooks) updateCredentials(ctx context.Context, id string, creds map[string]string) error {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	secret, err := s.loadSecret(ctx, id)
	if err != nil {
		return err
	}
	if secret == nil {
		return ErrUnknownSecret
	}

	secret.Data = map[string][]byte{}
	for key, value := range creds {
		secret.Data[key] = []byte(value)
	}

	// TODO use an ObjectManager
	return c.Update(ctx, secret)
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

func (s *servicesWebhooks) deleteCredentials(ctx context.Context, id string) error {
	secret, err := s.loadSecret(ctx, id)
	if err != nil {
		return err
	}
	if secret == nil {
		// nothing to do
		return nil
	}

	// the secret should already be marked for deletion, we just need to clear the finalizer
	return reconcilers.ClearFinalizer(ctx, secret, ServiceCredentialFinalizer)
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=list

func (s *servicesWebhooks) loadSecret(ctx context.Context, id string) (*corev1.Secret, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	secrets := &corev1.SecretList{}
	labelSelector := client.MatchingLabels(map[string]string{
		"services.wa8s.reconciler.io/id": id,
	})

	if err := c.List(ctx, secrets, labelSelector); err != nil {
		return nil, err
	}
	if len(secrets.Items) == 1 {
		return &secrets.Items[0], nil
	}

	// Try reading directly from the API in case the informer hasn't synced yet
	if err := c.APIReader.List(ctx, secrets, labelSelector); err != nil {
		return nil, err
	}
	if len(secrets.Items) == 1 {
		return &secrets.Items[0], nil
	}

	return nil, nil
}
