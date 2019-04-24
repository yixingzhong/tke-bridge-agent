package main

import (
	"net"
	"syscall"

	log "github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	mainRouteTable   = 254
	cidrRulePriority = 1024
)

func containsNoSuchRule(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.ENOENT
	}
	return false
}

func ensureRule(cidr *net.IPNet) error {
	log.Infof("Ensure rule %+v", cidr)

	rules, err := netlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return errors.Wrapf(err, "failed to list rule")
	}

	cidrStr := cidr.String()
	var found bool
	for _, rule := range rules {
		// only care rule pref == 1024
		if rule.Priority != cidrRulePriority {
			continue
		}
		if rule.Dst == nil {
			continue
		}
		if rule.Dst.String() != cidrStr {
			log.Infof("Clear extra rule (from %v to %v table %d)", rule.Src, rule.Dst, rule.Table)
			// clear extra rule
			err := netlink.RuleDel(&rule)
			if err != nil && !containsNoSuchRule(err) {
				return errors.Wrapf(err, "clear extra rule: failed to delete old rule %v", rule)
			}
		} else {
			log.Infof("skip add rule (from %v to %v table %d), same rule already exist", rule.Src, rule.Dst, rule.Table)
			found = true
		}
	}

	if !found {
		rule := netlink.NewRule()
		rule.Dst = cidr
		rule.Table = mainRouteTable
		rule.Priority = cidrRulePriority

		err = netlink.RuleAdd(rule)
		if err != nil {
			return errors.Wrapf(err, "add cidr rule: failed to add rule for %v", cidr)
		}
	}
	return nil
}
