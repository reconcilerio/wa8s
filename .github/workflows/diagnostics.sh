#!/bin/bash

set +o errexit
set -o nounset
set +o pipefail

echo "##[group]kubectl get all -n wa8s-system"
    kubectl get all -n wa8s-system || true
echo "##[endgroup]"
echo "##[group]kubectl get all -n wa8s-system -oyaml"
    kubectl get all -n wa8s-system -oyaml || true
echo "##[endgroup]"
echo "##[group]kubectl get wa8s --all-namespaces"
    kubectl get wa8s --all-namespaces || true
echo "##[endgroup]"
echo "##[group]kubectl get wa8s --all-namespaces -oyaml"
    kubectl get wa8s --all-namespaces -oyaml || true
echo "##[endgroup]"
echo "##[group]stern -n wa8s-system -l control-plane=wa8s-manager --tail 10000 --no-follow"
    ${STERN} -n wa8s-system -l control-plane=wa8s-manager --tail 10000 --no-follow || true
echo "##[endgroup]"
echo "##[group]stern -n wa8s-system -l control-plane=wa8s-knative-manager --tail 10000 --no-follow"
    ${STERN} -n wa8s-system -l control-plane=wa8s-knative-manager --tail 10000 --no-follow || true
echo "##[endgroup]"
echo "##[group]stern -n wa8s-system -l control-plane=wa8s-services-manager --tail 10000 --no-follow"
    ${STERN} -n wa8s-system -l control-plane=wa8s-services-manager --tail 10000 --no-follow || true
echo "##[endgroup]"
echo "##[group]stern -n reconcilerio-system -l control-plane=controller-manager --tail 10000 --no-follow"
    ${STERN} -n reconcilerio-system -l control-plane=controller-manager --tail 10000 --no-follow || true
echo "##[endgroup]"

echo "##[group]kubectl get all -n knative-serving"
    kubectl get all -n knative-serving || true
echo "##[endgroup]"
echo "##[group]kubectl get all -n knative-serving -oyaml"
    kubectl get all -n knative-serving -oyaml || true
echo "##[endgroup]"
echo "##[group]stern -n knative-serving -l app.kubernetes.io/name=knative-serving --tail 10000"
    ${STERN} -n knative-serving -l app.kubernetes.io/name=knative-serving --tail 10000 || true
echo "##[endgroup]"
echo "##[group]kubectl logs -n knative-serving -l app.kubernetes.io/name=knative-serving,app.kubernetes.io/component=controller --tail 10000 --previous"
    kubectl logs -n knative-serving -l app.kubernetes.io/name=knative-serving,app.kubernetes.io/component=controller --tail 10000 --previous || true
echo "##[endgroup]"
