# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: test

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=wa8s-manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	cat hack/boilerplate.yaml.txt > config/wa8s.yaml
	$(KUSTOMIZE) build config/default >> config/wa8s.yaml

.PHONY: generate
generate: components ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(DIEGEN) die:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(MAKE) fmt

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GOIMPORTS) --local reconciler.io/wa8s -w .

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: components
components: components/static-config.wasm components/wac.wasm components/wit-tools.wasm

components/static-config.wasm: $(shell find components/static-config -type f) $(shell find components/deps/static-config -not \( -path components/deps/static-config/target -prune \)) Cargo.toml
	$(shell cd components/deps/static-config && ./build.sh)
	cargo build -p static-config-extism --release --target wasm32-unknown-unknown
	@cp target/wasm32-unknown-unknown/release/static_config_extism.wasm components/static-config.wasm

components/wit-tools.wasm: $(shell find components/wit-tools -type f) Cargo.toml
	cargo build -p wit-tools --release --target wasm32-unknown-unknown
	@cp target/wasm32-unknown-unknown/release/wit_tools.wasm components/wit-tools.wasm

components/wac.wasm: $(shell find components/wac -type f) Cargo.toml
	cargo build -p wac --release --target wasm32-unknown-unknown
	@cp target/wasm32-unknown-unknown/release/wac.wasm components/wac.wasm

##@ Deployment

KAPP_APP ?= wa8s
KAPP_APP_NAMESPACE ?= default

ifeq (${KO_DOCKER_REPO},kind.local)
# kind isn't multi-arch aware, default to the current arch
KO_PLATFORMS ?= linux/$(shell go env GOARCH)
else
KO_PLATFORMS ?= linux/arm64,linux/amd64
endif

.PHONY: logs
logs: ## Watch logs from the wa8s-system namespace
	@$(STERN) -n wa8s-system .

.PHONY: logs-manager
logs-manager: ## Watch logs from the wa8s manager
	@$(STERN) -n wa8s-system -l control-plane=controller-manager

.PHONY: deploy
deploy: generate manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KAPP) deploy -a $(KAPP_APP) -n $(KAPP_APP_NAMESPACE) -c \
		-f config/kapp \
		-f <($(KO) resolve --platform $(KO_PLATFORMS) -f config/wa8s.yaml)

.PHONY: deploy-cert-manager
deploy-cert-manager: ## Deploy cert-manager to the K8s cluster specified in ~/.kube/config.
	$(KAPP) deploy -a cert-manager -n $(KAPP_APP_NAMESPACE) --wait-timeout 5m -c -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml

.PHONY: undeploy-cert-manager
undeploy-cert-manager: ## Undeploy cert-manager from the K8s cluster specified in ~/.kube/config.
	$(KAPP) delete -a cert-manager -n $(KAPP_APP_NAMESPACE)

.PHONY: deploy-ducks
deploy-ducks: ## Deploy ducks to the K8s cluster specified in ~/.kube/config.
	$(KAPP) deploy -a ducks -n $(KAPP_APP_NAMESPACE) --wait-timeout 5m -c -f https://github.com/reconcilerio/ducks/releases/download/v0.1.0/reconcilerio-ducks-v0.1.0.yaml

.PHONY: undeploy-ducks
undeploy-ducks: ## Undeploy cert-manager from the K8s cluster specified in ~/.kube/config.
	$(KAPP) delete -a ducks -n $(KAPP_APP_NAMESPACE)

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KAPP) delete -a $(KAPP_APP) -n $(KAPP_APP_NAMESPACE)

.PHONY: kind-deploy
kind-deploy: ## Deploy to a running local kind cluster
	KO_DOCKER_REPO=kind.local make deploy

##@ Dependencies

## Tool Binaries
CONTROLLER_GEN ?= go run -modfile hack/controller-gen/go.mod sigs.k8s.io/controller-tools/cmd/controller-gen
DIEGEN ?= go run -modfile hack/diegen/go.mod reconciler.io/dies/diegen
GOIMPORTS ?= go run -modfile hack/goimports/go.mod golang.org/x/tools/cmd/goimports
KAPP ?= go run -modfile hack/kapp/go.mod github.com/k14s/kapp/cmd/kapp
KO ?= go run -modfile hack/ko/go.mod github.com/google/ko
KUSTOMIZE ?= go run -modfile hack/kustomize/go.mod sigs.k8s.io/kustomize/kustomize/v4
STERN ?= go run -modfile hack/stern/go.mod github.com/stern/stern
