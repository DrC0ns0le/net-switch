package utils

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	perfutils "github.com/DrC0ns0le/net-perf/utils"
)

func EnableAsymmetricRoute() {

	keyValue := map[string]string{
		"net.ipv4.conf.all.accept_local":       "1",
		"net.ipv4.conf.all.route_localnet":     "1",
		"net.ipv4.conf.default.accept_local":   "1",
		"net.ipv4.conf.default.route_localnet": "1",
		"net.ipv4.conf.all.rp_filter":          "0",
		"net.ipv4.conf.default.rp_filter":      "0",
	}

	ifaces, err := perfutils.GetAllWGInterfaces()
	if err != nil {
		log.Panicf("failed to get interfaces: %v", err)
	}

	for _, iface := range ifaces {
		keyValue[fmt.Sprintf("net.ipv4.conf.%s.accept_local", strings.Replace(iface.Name, ".", "/", 1))] = "1"
		keyValue[fmt.Sprintf("net.ipv4.conf.%s.route_localnet", strings.Replace(iface.Name, ".", "/", 1))] = "1"
		keyValue[fmt.Sprintf("net.ipv4.conf.%s.rp_filter", strings.Replace(iface.Name, ".", "/", 1))] = "0"
	}

	for key, value := range keyValue {
		err := setSysctl(key, value)
		if err != nil {
			log.Panicf("failed to set sysctl %s: %v", key, err)
		}
	}
}

func setSysctl(name string, value string) error {
	return exec.Command("sysctl", "-w", fmt.Sprintf("%s=%s", name, value)).Run()
}
