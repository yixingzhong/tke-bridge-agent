package main

import (
	"net"
	"fmt"
	"path"
	"io/ioutil"
)

const (
	DefaultCniConfDir = "/host/etc/cni/net.d"
	pluginName = "tke-bridge"
)

const NET_CONFIG_TEMPLATE = `{
  "cniVersion": "0.1.0",
  "name": "tke-bridge",
  "type": "bridge",
  "bridge": "cbr0",
  "mtu": %d,
  "addIf": "eth0",
  "isGateway": true,
  "ipMasq": false,
  "hairpinMode": %t,
  "ipam": {
    "type": "host-local",
    "subnet": "%s",
    "gateway": "%s",
    "routes": [
      { "dst": "0.0.0.0/0" }
    ]
  }
}`

func generateBridgeConf(cidr *net.IPNet, mtu int, hairpinMode bool) error {
	subnet := cidr.String()
	cidr.IP[len(cidr.IP)-1] += 1
	gw := cidr.String()
	cniConf := fmt.Sprintf(NET_CONFIG_TEMPLATE, mtu, hairpinMode, subnet, gw)
	fileName := fmt.Sprintf("%s.conf", pluginName)
	return ioutil.WriteFile(path.Join(DefaultCniConfDir, fileName), []byte(cniConf), 0644)
}