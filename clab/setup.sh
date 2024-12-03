#!/bin/bash
#

sudo clab deploy --reconfigure --topo kind.clab.yml

docker image pull quay.io/metallb/frr-k8s:main
docker image pull quay.io/frrouting/frr:9.0.0
docker image pull quay.io/frrouting/frr:9.0.2
docker image pull gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1
kind load docker-image quay.io/frrouting/frr:9.0.0 --name k0
kind load docker-image quay.io/frrouting/frr:9.0.2 --name k0
kind load docker-image gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1 --name k0
kind load docker-image quay.io/metallb/frr-k8s:main --name k0

docker cp kind/setup.sh k0-control-plane:/setup.sh
docker exec k0-control-plane /setup.sh

kind --name k0 get kubeconfig > kubeconfig
export KUBECONFIG=$(pwd)/kubeconfig
kind/frr-k8s/setup.sh


sleep 4s
docker exec clab-kindpods-leaf1 /setup.sh
docker exec clab-kindpods-leaf2 /setup.sh
docker exec clab-kindpods-spine /setup.sh
docker exec clab-kindpods-HOST1 /setup.sh
