package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type Options struct {
	MTU         int
	HairpinMode string
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.MTU, "mtu", 0, "interface mtu")
	fs.StringVar(&o.HairpinMode, "hairpinMode", "promiscuous-bridge", `--hairpin-mode string How should the agent setup hairpin NAT. This allows endpoints of a Service to loadbalance back to themselves if they should try to access their own Service. Valid values are "promiscuous-bridge", "hairpin-veth" and "none". (default "promiscuous-bridge")`)

	return
}

func (o *Options) Validate() error {
	switch o.HairpinMode {
	case "promiscuous-bridge", "hairpin-veth", "none":
		return nil
	default:
		return errors.Errorf("invalid hairpin mode %s", o.HairpinMode)
	}
	return nil
}

func (o *Options) Config() error {
	if err := o.Validate(); err != nil {
		return err
	}
	return nil
}
