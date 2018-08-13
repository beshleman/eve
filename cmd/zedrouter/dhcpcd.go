// Copyright (c) 2018 Zededa, Inc.
// All rights reserved.

// Manage dhcpcd for uplinks including static
// XXX wwan0? Skip for now

package zedrouter

import (
	"fmt"
	"github.com/zededa/go-provision/types"
	"github.com/zededa/go-provision/wrap"
	"log"
	"reflect"
)

// Start/modify/delete dhcpcd per interface
func updateDhcpClient(newConfig, oldConfig types.DeviceUplinkConfig) {
	// Look for adds or changes
	for _, newU := range newConfig.Uplinks {
		oldU := lookupOnIfname(oldConfig, newU.IfName)
		if false {
			// XXX type check - remove
			*oldU = newU
		}
		if oldU == nil {
			doDhcpClientActivate(newU)
		} else if reflect.DeepEqual(newU, oldU) {
			doDhcpClientInactivate(*oldU)
			doDhcpClientActivate(newU)
		}
	}
	// Look for deletes from oldConfig to newConfig
	for _, oldU := range newConfig.Uplinks {
		newU := lookupOnIfname(newConfig, oldU.IfName)
		if newU == nil {
			doDhcpClientInactivate(oldU)
		}
	}

}

func lookupOnIfname(config types.DeviceUplinkConfig, ifname string) *types.NetworkUplinkConfig {
	for _, c := range config.Uplinks {
		if c.IfName == ifname {
			return &c
		}
	}
	return nil
}

// XXX determine static; determine -G
func doDhcpClientActivate(nuc types.NetworkUplinkConfig) {
	log.Printf("doDhcpClientActivate(%s) dhcp %v addr %s gateway %s\n",
		nuc.IfName, nuc.Dhcp, nuc.Addr.String(),
		nuc.Gateway.String())
	if nuc.IfName == "wwan0" {
		log.Printf("doDhcpClientActivate: skipping %s\n",
			nuc.IfName)
		return
	}
	ng := ""
	if nuc.Gateway.String() == "0.0.0.0" {
		ng = "--nogateway"
	}

	switch nuc.Dhcp {
	case types.DT_CLIENT:
		extras := []string{"-K", "--noipv4ll"}
		if ng != "" {
			extras = append(extras, ng)
		}
		if !dhcpcdCmd("--request", extras, nuc.IfName) {
			log.Printf("doDhcpClientActivate: request failed for %s\n",
				nuc.IfName)
		}
	case types.DT_STATIC:
		// XXX Addr vs. Subnet? Need netmask.
		args := []string{fmt.Sprintf("ip_addr=%s", nuc.Addr.String())}

		extras := []string{"-K", "--noipv4ll"}
		if ng != "" {
			extras = append(extras, ng)
		} else if nuc.Gateway.String() != "" {
			args = append(args, "--static",
				fmt.Sprintf("router=%s", nuc.Gateway.String()))
		}
		for _, dns := range nuc.DnsServers {
			args = append(args, "--static",
				fmt.Sprintf("dns=%s", dns.String()))
		}
		args = append(args, extras...)
		if !dhcpcdCmd("--static", args, nuc.IfName) {
			log.Printf("doDhcpClientActivate: request failed for %s\n",
				nuc.IfName)
		}
	default:
		log.Printf("doDhcpClientActivate: unsupported dhcp %v\n",
			nuc.Dhcp)
	}
}

func doDhcpClientInactivate(nuc types.NetworkUplinkConfig) {
	log.Printf("doDhcpClientInactivate(%s) dhcp %v addr %s gateway %s\n",
		nuc.IfName, nuc.Dhcp, nuc.Addr.String(),
		nuc.Gateway.String())
	if nuc.IfName == "wwan0" {
		log.Printf("doDhcpClientInactivate: skipping %s\n",
			nuc.IfName)
		return
	}
	extras := []string{"-K"}
	if !dhcpcdCmd("--release", extras, nuc.IfName) {
		log.Printf("doDhcpClientInactivate: release failed for %s\n",
			nuc.IfName)
	}
}

func dhcpcdCmd(op string, extras []string, ifname string) bool {
	cmd := "dhcpcd"
	args := append([]string{op}, extras...)
	args = append(args, ifname)
	if _, err := wrap.Command(cmd, args...).Output(); err != nil {
		return false
	}
	return true
}
