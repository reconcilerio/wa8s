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

package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CustomTransport(ctx context.Context) (http.RoundTripper, error) {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	// from github.com/google/go-containerregistry/pkg/v1/remote.DefaultTransport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// We usually are dealing with 2 hosts (at most), split MaxIdleConns between them.
		MaxIdleConnsPerHost: 50,
	}

	// customizations

	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	tlsSecrets := &corev1.SecretList{}
	// TODO parameterize the namespace
	if err := c.List(ctx, tlsSecrets, client.MatchingFields{"type": "kubernetes.io/tls"}, client.InNamespace("wa8s-system")); err != nil {
		return nil, err
	}
	for _, tlsSecret := range tlsSecrets.Items {
		if ca, ok := tlsSecret.Data["ca.crt"]; ok {
			rootCAs.AppendCertsFromPEM(ca)
		}
	}

	transport.TLSClientConfig = &tls.Config{
		RootCAs: rootCAs,
	}

	return transport, nil
}
