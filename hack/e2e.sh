#!/bin/sh

export OS_CLOUD=devstack-admin
openstack network set --name provider_net_dualstack_1 private

# Now drop admin privileges
export OS_CLOUD=devstack

sed "s/  devstack:/  openstack:/g" /etc/openstack/clouds.yaml > examples/bases/credentials/clouds.yaml
make --directory=examples/
cd examples/apply/cirros || exit
kubectl apply -k . --server-side
kubectl wait --timeout=10m --for=condition=available server ${USER}-cirros-server

openstack server show "$(kubectl get server ${USER}-cirros-server -o jsonpath='{.status.id}')"
