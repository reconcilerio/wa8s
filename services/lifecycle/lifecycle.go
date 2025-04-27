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

package lifecycle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
)

type ServiceInstanceId = string
type ServiceBindingId = string
type Request = servicesv1alpha1.ServiceInstanceRequest

type Context struct {
	metav1.OwnerReference `json:",inline"`
	Namespace             string `json:"namespace"`
}

func encodeContext(ctx context.Context, owner client.Object) string {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	gvk, err := c.Client.GroupVersionKindFor(owner)
	if err != nil {
		panic(err)
	}

	context := Context{
		OwnerReference: metav1.OwnerReference{
			APIVersion:         gvk.GroupVersion().String(),
			Kind:               gvk.Kind,
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			BlockOwnerDeletion: ptr.To(true),
			Controller:         ptr.To(true),
		},
		Namespace: owner.GetNamespace(),
	}
	data, err := json.Marshal(context)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func ParseContext(str string) (Context, error) {
	data, err := base64.RawURLEncoding.DecodeString(str)
	if err != nil {
		return Context{}, err
	}
	var context Context
	if err := json.Unmarshal(data, &context); err != nil {
		return Context{}, err
	}
	return context, nil
}

type lifecycle struct {
	address string
}

func NewLifecycle(address string) *lifecycle {
	return &lifecycle{
		address: address,
	}
}

func (l *lifecycle) Provision(ctx context.Context, instanceId ServiceInstanceId, type_ string, tier *string, requests []Request) error {
	u, err := url.Parse(l.address)
	if err != nil {
		return err
	}
	u.Path = "/provision"
	q := u.Query()
	q.Set("instance-id", instanceId)
	q.Set("type", type_)
	if tier != nil {
		q.Set("tier", *tier)
	}
	for _, r := range requests {
		q.Add("request", fmt.Sprintf("%s=%s", r.Key, r.Value))
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("server returned http %d", resp.StatusCode)
	}
	return nil
}

func (l *lifecycle) Destroy(ctx context.Context, instanceId ServiceInstanceId, retain *bool) error {
	u, err := url.Parse(l.address)
	if err != nil {
		return err
	}
	u.Path = "/destroy"
	q := u.Query()
	q.Set("instance-id", instanceId)
	if retain != nil {
		q.Set("retain", fmt.Sprintf("%t", *retain))
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("server returned http %d", resp.StatusCode)
	}
	return nil
}

func (l *lifecycle) Bind(ctx context.Context, bindingId ServiceBindingId, instanceId ServiceInstanceId, scopes []string) error {
	u, err := url.Parse(l.address)
	if err != nil {
		return err
	}
	u.Path = "/bind"
	q := u.Query()
	q.Set("binding-id", bindingId)
	q.Set("instance-id", instanceId)
	for _, scope := range scopes {
		q.Add("scopes", scope)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("server returned http %d", resp.StatusCode)
	}
	return nil
}

func (l *lifecycle) Unbind(ctx context.Context, bindingId ServiceBindingId, instanceId ServiceInstanceId) error {
	u, err := url.Parse(l.address)
	if err != nil {
		return err
	}
	u.Path = "/unbind"
	q := u.Query()
	q.Set("binding-id", bindingId)
	q.Set("instance-id", instanceId)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("server returned http %d", resp.StatusCode)
	}
	return nil
}
