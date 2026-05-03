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

package registry

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

func KeychainForRepo(ctx context.Context, repo registriesv1alpha1.GenericRepository) (authn.Keychain, error) {
	return KeychainForServiceAccountRef(ctx, repo.GetSpec().ServiceAccountRef)
}

func KeychainForServiceAccountRef(ctx context.Context, serviceAccountRef registriesv1alpha1.ServiceAccountReference) (authn.Keychain, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: serviceAccountRef.Namespace,
			Name:      serviceAccountRef.Name,
		},
	}
	if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(serviceAccount), serviceAccount); err != nil {
		return nil, err
	}

	pullSecrets := []corev1.Secret{}
	for _, ref := range serviceAccount.ImagePullSecrets {
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: serviceAccount.Namespace,
				Name:      ref.Name,
			},
		}
		if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(pullSecret), pullSecret); err != nil {
			return nil, err
		}
		pullSecrets = append(pullSecrets, *pullSecret)
	}

	keychain, err := k8schain.NewFromPullSecrets(ctx, pullSecrets)
	if err != nil {
		return nil, err
	}

	return keychain, nil
}
