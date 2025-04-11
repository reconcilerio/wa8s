# Reconciler.io WebAssemblies for Kubernetes (wa8s) <!-- omit in toc -->

![CI](https://github.com/reconcilerio/wa8s/workflows/CI/badge.svg?branch=main)
[![GoDoc](https://godoc.org/reconciler.io/wa8s?status.svg)](https://godoc.org/reconciler.io/wa8s)
[![Go Report Card](https://goreportcard.com/badge/reconciler.io/wa8s)](https://goreportcard.com/report/reconciler.io/wa8s)
[![codecov](https://codecov.io/gh/reconcilerio/wa8s/main/graph/badge.svg)](https://codecov.io/gh/reconcilerio/wa8s)

`wa8s` is an experimental library for interacting with WebAssembly Components in a Kubernetes cluster.

- [Overview](#overview)
  - [Components](#components)
  - [Registries](#registries)
  - [Containers](#containers)
- [Getting Started](#getting-started)
  - [Deploy a released build](#deploy-a-released-build)
  - [Build from source](#build-from-source)
    - [Undeploy controller](#undeploy-controller)
- [Community](#community)
  - [Code of Conduct](#code-of-conduct)
  - [Communication](#communication)
  - [Contributing](#contributing)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Overview

### Components

The `components` package contains the core building blocks to resolve, compose, publish and create static components.

### Registries

The `registries` package defines OCI repositories where components can be published. 

By default an authenticated `ClusterRepository` is provided. It's hosted in cluster as a scratch space for intermediate components. State in this repository is ephemerial. Components must be published to a durable repository before they are consumed outside of wa8s.

### Containers

The `containers` package converts a component into a traditional OCI container, typically with [Wasmtime](https://wasmtime.dev) embedded. Using containers is a compatibility layer to run within environments that don't natively support wasm components.

Running a container requires publishing the container image to a repository the cluster is able to pull from. The default registry is not accessible to the cluster.

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [kind](https://kind.sigs.k8s.io) to get a local cluster for testing, or run against a remote cluster.

[`cert-manager`](https://cert-manager.io) and [`reconciler.io/ducks`](https://github.com/reconcilerio/ducks) are required in the cluster. They can be installed via `make deploy-cert-manager` and `make deploy-ducks` for default configurations.

### Deploy a released build

The easiest way to get started is by deploying the [latest release](https://github.com/reconcilerio/wa8s/releases). Alternatively, you can [build from source](#build-from-source).

### Build from source

1. Define where to publish images:

   ```sh
   export KO_DOCKER_REPO=<a-repository-you-can-write-to>
   ```

   For kind, a registry is not required (or run `make kind-deploy`):

   ```sh
   export KO_DOCKER_REPO=kind.local
   ```
	
1. Build and deploy the controller to the cluster:

   Note: The cluster must have the [cert-manager](https://cert-manager.io) deployed.  There is a `make deploy-cert-manager` target to deploy the cert-manager.

   ```sh
   make deploy
   ```

#### Undeploy controller
Undeploy the controller to the cluster:

```sh
make undeploy
```


## Community

### Code of Conduct

The reconciler.io projects follow the [Contributor Covenant Code of Conduct](./CODE_OF_CONDUCT.md). In short, be kind and treat others with respect.

### Communication

General discussion and questions about the project can occur either on the Kubernetes Slack [#reconcilerio](https://kubernetes.slack.com/archives/C07J5G9NDHR) channel, or in the project's [GitHub discussions](https://github.com/orgs/reconcilerio/discussions). Use the channel you find most comfortable.

### Contributing

The reconciler.io wa8s project team welcomes contributions from the community. A contributor license agreement (CLA) is not required. You own full rights to your contribution and agree to license the work to the community under the Apache License v2.0, via a [Developer Certificate of Origin (DCO)](https://developercertificate.org). For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## Acknowledgements

This project was conceived in discussion with [Mark Fisher](https://github.com/markfisher).

## License

Apache License v2.0: see [LICENSE](./LICENSE) for details.
