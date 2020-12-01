#!/usr/bin/env sh

echo "=====Starting installing TKE-CNI ========="
cp /tke-route-eni /host/opt/cni/bin/
echo "=====Starting tke-bridge-agent ==========="
/tke-bridge-agent $@