#!/bin/bash

REGISTRY=${REGISTRY:-quay.io/orc}
IMAGE=${BASE_IMAGE:-openstack-resource-controller}
TAG=${BASE_IMAGE:-$(git describe --abbrev=0 --tags)}
IMG=${REGISTRY}/${IMAGE}:${TAG}

# Update config/manifests/bases/orc.clusterserviceversion.yaml if needed
operator-sdk generate kustomize manifests -q --plugins=go.kubebuilder.io/v4

# Create an overlay for customizing the controller image
TMP_OVERLAY=config/manifests_overlay
mkdir "${TMP_OVERLAY}"
pushd "${TMP_OVERLAY}" || exit
kustomize create --resources ../manifests
kustomize edit set image controller="${IMG}"

kustomize edit add patch --kind ClusterServiceVersion --name "orc.*" --patch '[{"op": "replace", "path": "/spec/version", "value": "'$TAG'"}]'
kustomize edit add patch --kind ClusterServiceVersion --name "orc.*" --patch '[{"op": "replace", "path": "/metadata/name", "value": "orc.'$TAG'"}]'
popd || exit

# Generate bundle and bundle.Dockerfile
kustomize build "${TMP_OVERLAY}" | operator-sdk generate bundle --plugins=go.kubebuilder.io/v4 --use-image-digests

rm -rf "${TMP_OVERLAY}"
