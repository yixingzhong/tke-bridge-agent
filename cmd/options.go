package main

import (
	"github.com/spf13/pflag"
)

type Options struct {
	MTU int
	HairpinMode bool
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.MTU, "mtu", 1460, "interface mtu")
	fs.BoolVar(&o.HairpinMode, "hairpinMode", true, "Whether bridge create in hairpinMode.")

	return
}
