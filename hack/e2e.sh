#!/bin/sh

export OS_CLOUD=devstack

sed "s/  devstack:/  openstack:/g" /etc/openstack/clouds.yaml > examples/credentials/clouds.yaml
kubectl apply -k examples/credentials-only --server-side
kubectl apply -k examples/centos-stream --server-side
kubectl wait --timeout=10m --for=condition=available image centos-stream-9

openstack image show "$(kubectl get image centos-stream-9 -o jsonpath='{.status.id}')"
