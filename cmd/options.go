package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Options struct {
	MTU              int
	HairpinMode      string
	AddRule          bool
	CniConfDir       string
	PortMapping      bool
	Bandwidth        bool
	AllocateInfoPath string
}

func NewOptions() *Options {
	return &Options{
		MTU:              0,
		HairpinMode:      "promiscuous-bridge",
		AddRule:          true,
		CniConfDir:       defaultCniConfDir,
		PortMapping:      true,
		Bandwidth:        false,
		AllocateInfoPath: "",
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.MTU, "mtu", o.MTU, "interface mtu")
	fs.StringVar(&o.HairpinMode, "hairpin-mode", o.HairpinMode, `--hairpin-mode string How should the agent setup hairpin NAT. This allows endpoints of a Service to loadbalance back to themselves if they should try to access their own Service. Valid values are "promiscuous-bridge", "hairpin-veth" and "none".`)
	fs.BoolVar(&o.AddRule, "add-rule", o.AddRule, `--add-rule bool whether add rule or not`)
	fs.StringVar(&o.CniConfDir, "cni-conf-dir", o.CniConfDir, `--cni-conf-dir string where tke-bridge.conf located`)
	fs.BoolVar(&o.PortMapping, "port-mapping", o.PortMapping, `--port-mapping bool whether support port-mapping or not`)
	fs.BoolVar(&o.Bandwidth, "bandwidth", o.Bandwidth, `--bandwidth bool whether support bandwidth or not`)
	fs.StringVar(&o.AllocateInfoPath, "allocateInfoPath", "", "--allocateInfoPath string where the ip allocate info located")
	return
}

func (o *Options) Validate() error {
	if o.CniConfDir == "" {
		return errors.New("cni-conf-dir cannot be empty")
	}
	switch o.HairpinMode {
	case "promiscuous-bridge", "hairpin-veth", "none":
		return nil
	default:
		return errors.Errorf("invalid hairpin mode %s", o.HairpinMode)
	}
}

func (o *Options) Config() error {
	if err := o.Validate(); err != nil {
		return err
	}
	return nil
}
