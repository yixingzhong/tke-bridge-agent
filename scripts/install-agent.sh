#!/usr/bin/env sh
echo "=====Starting installing cni ========="
tar xzf /cni-plugins.tgz -C /host/opt/cni/bin/
echo "=====Starting tke-bridge-agent ==========="
/tke-bridge-agent