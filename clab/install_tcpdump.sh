#!/bin/bash

kubectl get pod \
    -l app=router \
    -n openperouter-system \
    --field-selector=status.phase=Running \
    -o custom-columns=name:metadata.name --no-headers \
    | xargs -I{} kubectl -n openperouter-system exec {} apk add tcpdump

