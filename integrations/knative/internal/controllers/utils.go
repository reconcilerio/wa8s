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
	"reconciler.io/wa8s/controllers"
)

var (
	// ErrTransient captures an error that is of the moment, retrying the request may succeed. Meaningful state about the error has been captured on the status
	ErrTransient = controllers.ErrTransient
	// ErrDurable is permanent given the current state, the request should not be retried until the observed state has changed. Meaningful state about the error has been captured on the status
	ErrDurable = controllers.ErrDurable
	// ErrGenerationMismatch a referenced resource's .metadata.generation and .status.observedGeneration are out of sync. Treat as a transient error as this state is expected and we should avoid flapping
	ErrGenerationMismatch = controllers.ErrGenerationMismatch
	// ErrUpdateStatusBeforeContinuingReconcile halt this reconcile request and update the api server with the intermediate status
	ErrUpdateStatusBeforeContinuingReconcile = controllers.ErrUpdateStatusBeforeContinuingReconcile
)
