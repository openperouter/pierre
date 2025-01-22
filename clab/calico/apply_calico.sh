#!/bin/bash

timeout=600
start_time=$(date +%s)

# Function to check elapsed time
check_timeout() {
    local current_time=$(date +%s)
    local elapsed=$((current_time - start_time))
    if [ "$elapsed" -ge "$timeout" ]; then
        echo "Timeout reached after $timeout seconds."
        exit 1
    fi
}

# Loop until all commands succeed or timeout
while true; do
    sleep 5 # Optional: Add a delay between retries to avoid excessive CPU usage
    check_timeout # Check timeout after each command
    echo "Apply calico."
    if ! kind get kubeconfig --name pe-kind > kubeconfig; then
	    continue
    fi
    export KUBECONFIG=./kubeconfig
    if ! kubectl apply --server-side=true -f tigera-operator.yaml; then
	    continue
    fi

    if ! kubectl apply --server-side=true -f calico-config.yaml; then
	    continue
    fi

    echo "Apply calico succeeded."
    exit 0

done

