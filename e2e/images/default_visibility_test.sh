#!/usr/bin/env bash

set -Eeuo pipefail
set -x

declare -r image_name='e2e-images-visibility'

cat <<EOF | kubectl apply -f -
apiVersion: openstack.gopherkube.dev/v1alpha1
kind: OpenStackImage
metadata:
  labels:
    app.kubernetes.io/name: openstackimage
  name: ${image_name}
spec:
  cloud: osp1
  name: ${image_name}
EOF

kubectl wait --timeout=10s --for=jsonpath='{.status.name}'="${image_name}" OpenStackImage "${image_name}"

declare visibility=''
visibility="$(openstack image show "$(kubectl get OpenStackImage "${image_name}" -o=jsonpath='{.status.id}')" -f value -c visibility)"

kubectl delete OpenStackImage "${image_name}"

[[ "$visibility" == 'shared' ]]
