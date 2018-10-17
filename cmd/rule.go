package main

import (
	"net"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	mainRouteTable = 254
	cidrRulePriority = 1024
)

func containsNoSuchRule(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.ENOENT
	}
	return false
}

func ensureRule(cidr *net.IPNet) error {
	err := clearPreviousRule()
	if err != nil {
		return err
	}

	rule := netlink.NewRule()
	rule.Dst = cidr
	rule.Table = mainRouteTable
	rule.Priority = cidrRulePriority

	err = netlink.RuleAdd(rule)
	if err != nil {
		return errors.Wrapf(err, "add cidr rule: failed to add rule for %v", *cidr)
	}
	return nil
}

func clearPreviousRule() error {
	rule := netlink.NewRule()
	rule.Table = mainRouteTable
	rule.Priority = cidrRulePriority

	err := netlink.RuleDel(rule)
	if err != nil && !containsNoSuchRule(err) {
		return errors.Wrapf(err, "clear previous rule: failed to delete old rule",)
	}
	return nil
}