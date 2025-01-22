#!/bin/bash

run_and_echo() {
    clear
    local comment="$1"
    shift

    local cmd="$*"

    echo -e "\n$comment\n\n"
    echo -e "Executing: $cmd\n"

    eval "$cmd"

    echo ""
    read
}


run_and_echo "kind cluster with two nodes" kubectl get nodes

run_and_echo "calico configured to peer with the perouter. All nodes use the same IP" kubectl get bgppeer -o yaml

run_and_echo "the status is not established yet" kubectl get caliconodestatus status -o jsonpath='{.status.bgp}'

openpepod=$(kubectl get pods -n openperouter-system -l app=router | awk 'NR==2 {print $1}')

run_and_echo "the frr container does not have extra interfaces" "kubectl exec -n openperouter-system $openpepod -c frr -- ip l"

run_and_echo "no routes for pods" "docker exec pe-kind-worker ip r show | grep hostred"

run_and_echo "workload pods" kubectl get pods -o wide

source=$(kubectl get pods -l app=workload | awk 'NR==2 {print $1}')
target_pod_ip=$(kubectl get pods -o wide -l app=workload | awk 'NR==3 {print $6}')

# Pinging the second pod from the first

run_and_echo "pinging from one pod to the other" "kubectl exec $source -- ping -c 1 $target_pod_ip"

clear
echo "applying openperouter config under config/samples"
sleep 1s
kubectl apply -f config/samples/vni.yaml
kubectl apply -f config/samples/underlay.yaml
sleep 3s

echo "waiting for calico peer status to be established"

while :; do
    peerstatus=$(kubectl get caliconodestatuses status -o=jsonpath='{.status.bgp.peersV4[0].state}')
    if echo "$peerstatus" | grep -q "Established"; then
        echo "Session is established"
        break
    fi
    sleep 1
done

sleep 1s

run_and_echo "status is now established" kubectl get caliconodestatus status -o jsonpath='{.status.bgp}'

run_and_echo "now the frr container has the interfaces for vxlan" "kubectl exec -n openperouter-system $openpepod -c frr -- ip l"

run_and_echo "check the cidrs for each node" "kubectl get ipamblocks"

run_and_echo "the worker node has a route to the pods cidr on the other node" "docker exec pe-kind-worker ip r show | grep hostred"
run_and_echo "same for pod cidr of worker node" "docker exec pe-kind-control-plane ip r show | grep hostred"


run_and_echo "workload pods" kubectl get pods -o wide

run_and_echo "pinging from one pod to the other" "kubectl exec $source -- ping -c 1 $target_pod_ip"
run_and_echo "pinging from the host to the pod" "docker exec clab-kind-HOST1 ping -c 1 $target_pod_ip"

clear
echo -e "Let's repeat while tcpdumping inside a perouter pod.\n\n"

{ kubectl exec  -n openperouter-system $openpepod -c frr -- timeout 5s tcpdump -nn -i any icmp or udp > >(cat > output1) & } ; { sleep 2s && kubectl exec $source -- ping -c 1 $target_pod_ip > >(cat > output2) & } ; wait
echo ""
cat output2
echo ""
cat output1
