package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"

	"github.com/containernetworking/plugins/pkg/ip"
	log "github.com/golang/glog"
)

const (
	defaultCniConfDir = "/host/etc/cni/net.d"
	pluginName        = "tke-bridge"
	bridgeName        = "cbr0"
)

const NET_CONFIG_TEMPLATE = `{
  "cniVersion": "0.3.1",
  "name": "tke-bridge",
  "plugins": [
    {
      "cniVersion": "0.1.0",
      "type": "bridge",
      "bridge": "%s",
      "mtu": %d,
      "addIf": "eth0",
      "isGateway": true,
      "forceAddress": true,
      "ipMasq": false,
      "hairpinMode": %t,
      "promiscMode": %t,
      "ipam": {
        "type": "host-local",
        "subnet": "%s",
        "gateway": "%s",
        "routes": [
          {
            "dst": "0.0.0.0/0"
          }
        ]
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      },
      "externalSetMarkChain": "KUBE-MARK-MASQ"
    }
  ]
}`

// Enum settings for different ways to handle hairpin packets.
const (
	// Set the hairpin flag on the veth of containers in the respective
	// container runtime.
	HairpinVeth = "hairpin-veth"
	// Make the container bridge promiscuous. This will force it to accept
	// hairpin packets, even if the flag isn't set on ports of the bridge.
	PromiscuousBridge = "promiscuous-bridge"
	// Neither of the above. If the kubelet is started in this hairpin mode
	// and kube-proxy is running in iptables mode, hairpin packets will be
	// dropped by the container bridge.
	HairpinNone = "none"
)

func generateBridgeConf(cidr *net.IPNet, mtu int, hairpinMode string, confDir string) error {
	subnet := cidr.String()
	ipn := cidr.IP.Mask(cidr.Mask)
	gw := ip.NextIP(ipn).String()

	var iMtu int
	if mtu == 0 {
		if link, err := findMinMTU(); err == nil {
			iMtu = link.MTU
			log.Infof("Using interface %s MTU %d as bridge MTU", link.Name, link.MTU)
		} else {
			iMtu = 1460
			log.Warningf("Failed to find default bridge MTU, using %d: %v", iMtu, err)
		}
	} else {
		iMtu = mtu
	}

	var bHairpinMode, bPromiscMode bool
	switch hairpinMode {
	case HairpinVeth:
		bHairpinMode = true
		bPromiscMode = false
	case PromiscuousBridge:
		bHairpinMode = false
		bPromiscMode = true
	default:
		bHairpinMode = false
		bPromiscMode = false
	}

	cniConf := fmt.Sprintf(NET_CONFIG_TEMPLATE, bridgeName, iMtu, bHairpinMode, bPromiscMode, subnet, gw)
	fileName := fmt.Sprintf("20-%s.conflist", pluginName)
	log.Infof("Generate bridge conf %s : %s", fileName, cniConf)

	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		if err1 := os.Mkdir(confDir, 0755); err1 != nil {
			return err1
		}
	}

	return ioutil.WriteFile(path.Join(confDir, fileName), []byte(cniConf), 0644)
}

func findMinMTU() (*net.Interface, error) {
	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	mtu := 999999
	defIntfIndex := -1
	for i, intf := range intfs {
		if ((intf.Flags & net.FlagUp) != 0) && (intf.Flags&(net.FlagLoopback|net.FlagPointToPoint) == 0) {
			if intf.MTU < mtu {
				mtu = intf.MTU
				defIntfIndex = i
			}
		}
	}

	if mtu >= 999999 || mtu < 576 || defIntfIndex < 0 {
		return nil, fmt.Errorf("no suitable interface: %v", bridgeName)
	}

	return &intfs[defIntfIndex], nil
}
