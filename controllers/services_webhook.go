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
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"reconciler.io/wa8s/services/lifecycle"
)

type CredentialType = string

const (
	InstanceCredentials CredentialType = "instance"
	BindingCredentials  CredentialType = "binding"
)

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
	log, err := logr.FromContext(ctx)
	if err != nil {
		log = s.mgr.GetLogger()
	}
	log = log.WithName("ServicesWebhook")
	ctx = logr.NewContext(ctx, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/services/ids/instance", s.generateInstanceID(ctx))
	mux.HandleFunc("/services/ids/binding", s.generateBindingID(ctx))
	mux.HandleFunc("/services/ids/lookup", s.lookupInstanceID(ctx))
	mux.HandleFunc("/services/credentials/fetch", s.fetchCredentials(ctx))
	mux.HandleFunc("/services/credentials/publish", s.publishCredentials(ctx))
	mux.HandleFunc("/services/credentials/destroy", s.destroyCredentials(ctx))

	return http.ListenAndServe(s.Addr, mux)
}

func (s *servicesWebhooks) generateInstanceID(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("generateInstanceID")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI, "context", r.Header.Get("context"))

		context, err := lifecycle.ParseContext(r.Header.Get("context"))
		if err != nil {
			log.Error(err, "context missing")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		instanceId, err := s.createInstanceId(ctx, context)
		if err != nil {
			log.Error(err, "failed to create service-instance-id")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("service-instance-id", instanceId)
		w.WriteHeader(http.StatusNoContent)
		log.Info("response", "service-instance-id", instanceId)
	}
}

func (s *servicesWebhooks) generateBindingID(ctx context.Context) func(w http.ResponseWriter, r *http.Request) {
	log := logr.FromContextOrDiscard(ctx).WithName("generateBindingID")
	ctx = logr.NewContext(ctx, log)

	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("request", "method", r.Method, "path", r.RequestURI, "context", r.Header.Get("context"))

		context, err := lifecycle.ParseContext(r.Header.Get("context"))
		if err != nil {
			log.Error(err, "context missing")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		instanceId := r.Header.Get("service-instance-id")
		if instanceId == "" {
			log.Error(fmt.Errorf("missing service-instance-id header"), "missing service-instance-id header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bindingId, err := s.createBindingCredentials(ctx, context, instanceId)
		if err != nil {
			log.Error(err, "failed to create service-binding-id")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("service-binding-id", bindingId)
		w.WriteHeader(http.StatusNoContent)
		log.Info("response", "service-binding-id", bindingId)
	}
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

		secret, err := s.loadSecret(ctx, BindingCredentials, bindingId)
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

		creds, err := s.retrieveCredentials(ctx, BindingCredentials, bindingId)
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

		err := s.updateCredentials(ctx, BindingCredentials, bindingId, creds)
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

		err := s.deleteCredentials(ctx, BindingCredentials, bindingId)
		if err != nil {
			log.Error(err, "failed to destroy service-binding-id")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=create

func (s *servicesWebhooks) createInstanceId(ctx context.Context, context lifecycle.Context) (string, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)
	log := logr.FromContextOrDiscard(ctx)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: context.Namespace,
			Name:      fmt.Sprintf("service-%s-%s", InstanceCredentials, context.UID),
			Labels: map[string]string{
				"services.wa8s.reconciler.io/type":                                    InstanceCredentials,
				fmt.Sprintf("services.wa8s.reconciler.io/%s-id", InstanceCredentials): string(context.UID),
			},
			OwnerReferences: []metav1.OwnerReference{context.OwnerReference},
		},
	}

	// dry run will assign a UID, without creating a resource
	if err := c.Create(ctx, secret); err != nil {
		if apierrs.IsAlreadyExists(err) {
			log.Error(nil, "attempted to recreate instance", "instanceId", context.UID, "err", err)
			return string(context.UID), nil
		}
		return "", err
	}

	return string(context.UID), nil
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=create

func (s *servicesWebhooks) createBindingCredentials(ctx context.Context, context lifecycle.Context, instanceId string) (string, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)
	log := logr.FromContextOrDiscard(ctx)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: context.Namespace,
			Name:      fmt.Sprintf("service-%s-%s", BindingCredentials, context.UID),
			Labels: map[string]string{
				"services.wa8s.reconciler.io/type":                                    BindingCredentials,
				fmt.Sprintf("services.wa8s.reconciler.io/%s-id", BindingCredentials):  string(context.UID),
				fmt.Sprintf("services.wa8s.reconciler.io/%s-id", InstanceCredentials): instanceId,
			},
			OwnerReferences: []metav1.OwnerReference{context.OwnerReference},
		},
	}
	if err := c.Create(ctx, secret); err != nil {

		if apierrs.IsAlreadyExists(err) {
			log.Error(nil, "attempted to recreate binding", "instanceId", instanceId, "bindingId", context.UID, "err", err)
			return string(context.UID), nil
		}
		return "", err
	}

	return string(context.UID), nil
}

func (s *servicesWebhooks) retrieveCredentials(ctx context.Context, secretType CredentialType, id string) (map[string]string, error) {
	secret, err := s.loadSecret(ctx, secretType, id)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, fmt.Errorf("unknown secret")
	}

	credentials := map[string]string{}
	for key, value := range secret.Data {
		credentials[key] = string(value)
	}

	return credentials, nil
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=update

func (s *servicesWebhooks) updateCredentials(ctx context.Context, secretType CredentialType, id string, creds map[string]string) error {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	secret, err := s.loadSecret(ctx, secretType, id)
	if err != nil {
		return err
	}
	if secret == nil {
		return fmt.Errorf("unknown secret")
	}

	secret.Data = map[string][]byte{}
	for key, value := range creds {
		secret.Data[key] = []byte(value)
	}

	return c.Update(ctx, secret)
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=delete

func (s *servicesWebhooks) deleteCredentials(ctx context.Context, secretType CredentialType, id string) error {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	secret, err := s.loadSecret(ctx, secretType, id)
	if err != nil {
		return err
	}
	if secret == nil {
		// nothing to do
		return nil
	}

	return c.Delete(ctx, secret)
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=list

func (s *servicesWebhooks) loadSecret(ctx context.Context, secretType CredentialType, id string) (*corev1.Secret, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	secrets := &corev1.SecretList{}
	labelSelector := client.MatchingLabels(map[string]string{
		"services.wa8s.reconciler.io/type":                           secretType,
		fmt.Sprintf("services.wa8s.reconciler.io/%s-id", secretType): id,
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
