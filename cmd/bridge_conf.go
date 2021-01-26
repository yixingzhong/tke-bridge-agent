package main

import (
	"encoding/json"
	"fmt"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/plugins/pkg/ip"
	"io/ioutil"
	"net"
	"os"
	"path"

	log "github.com/golang/glog"
)

const (
	defaultCniConfDir = "/host/etc/cni/net.d"
	pluginName        = "tke-bridge"
	bridgeName        = "cbr0"
)

const NET_CONFIG_TEMPLATE = `{
  "cniVersion": "0.1.0",
  "name": "tke-bridge",
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
      { "dst": "0.0.0.0/0" }
    ]
  }
}`

type RangeSet []Range

type conf struct {
	CniVersion   string     `json:"cniVersion"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	RouteTable   int        `json:"routeTable"`
	Bridge       string     `json:"bridge"`
	Mtu          int        `json:"mtu"`
	AddIf        string     `json:"addIf"`
	IsGateway    bool       `json:"isGateway"`
	ForceAddress bool       `json:"forceAddress"`
	HairPinMode  bool       `json:"hairpinMode"`
	PromisecMode bool       `json:"promiscMode"`
	Ipam         IPAMConfig `json:"ipam"`
}

type IPAMConfig struct {
	*Range
	Name       string         `json:"name,omitempty"`
	Type       string         `json:"type"`
	Routes     []*types.Route `json:"routes"`
	DataDir    string         `json:"dataDir,omitempty"`
	ResolvConf string         `json:"resolvConf,omitempty"`
	Ranges     []RangeSet     `json:"ranges"`
	IPArgs     []net.IP       `json:"-"` // Requested IPs from CNI_ARGS and args
}

type Range struct {
	RangeStart net.IP      `json:"rangeStart,omitempty"` // The first ip, inclusive
	RangeEnd   net.IP      `json:"rangeEnd,omitempty"`   // The last ip, inclusive
	Subnet     types.IPNet `json:"subnet"`
	Gateway    net.IP      `json:"gateway,omitempty"`
}

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

// 给存量节点添加配置文件
func generateOldBridgeConf(cidr *net.IPNet, mtu int, hairpinMode string, confDir string) error {
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
	fileName := fmt.Sprintf("%s.conf", pluginName)
	log.Infof("Generate bridge conf %s : %s", fileName, cniConf)

	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		if err1 := os.Mkdir(confDir, 0755); err1 != nil {
			return err1
		}
	}

	return ioutil.WriteFile(path.Join(confDir, fileName), []byte(cniConf), 0644)
}

// 给增量节点添加配置文件
func generateNewBridgeConf(podCIDRs []*net.IPNet, mtu int, hairpinMode string, confDir string) error {
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

	var config conf
	config.Name = "tke-bridge"
	config.CniVersion = "0.1.0"
	config.Type = "tke-route-eni"
	config.RouteTable = -1
	config.Bridge = bridgeName
	config.Mtu = iMtu
	config.AddIf = "eth0"
	config.IsGateway = true
	config.ForceAddress = true
	config.HairPinMode = bHairpinMode
	config.PromisecMode = bPromiscMode
	config.Ipam.Type = "host-local"
	_, dst, _ := net.ParseCIDR("0.0.0.0/0")
	config.Ipam.Routes = []*types.Route{{Dst: *dst}}
	config.Ipam.Ranges = make([]RangeSet, 1)
	config.Ipam.Ranges[0] = make([]Range, len(podCIDRs))
	for idx, cidr := range podCIDRs {
		ipn := cidr.IP.Mask(cidr.Mask)
		config.Ipam.Ranges[0][idx] = Range{Gateway: ipn, Subnet: types.IPNet(*cidr)}
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}

	//cniConf := fmt.Sprintf(NET_CONFIG_TEMPLATE, bridgeName, iMtu, bHairpinMode, bPromiscMode, subnet, gw)
	fileName := fmt.Sprintf("%s.conf", pluginName)
	log.Infof("Generate bridge conf %s : %s", fileName, string(configJson))

	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		if err1 := os.Mkdir(confDir, 0755); err1 != nil {
			return err1
		}
	}

	return ioutil.WriteFile(path.Join(confDir, fileName), configJson, 0644)
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
