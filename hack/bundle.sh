#!/bin/bash
#http://quay.io/orc/openstack-resource-controller:v2.2.0

REGISTRY=${REGISTRY:-quay.io/orc}
IMAGE=${BASE_IMAGE:-openstack-resource-controller}
TAG=${BASE_IMAGE:-v2.2.0}
IMG=${REGISTRY}/${IMAGE}:${TAG}

operator-sdk generate kustomize manifests -q --plugins=go.kubebuilder.io/v4
cd config/manager && kustomize edit set image controller=${IMG} && cd ../..
kustomize build config/manifests | operator-sdk generate bundle --plugins=go.kubebuilder.io/v4 --use-image-digests
