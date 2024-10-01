package utils

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
)

type Route struct {
	Destination net.IP
	Gateway     net.IP
	Flags       int
	Iface       string
	Mask        net.IPMask
}

func GetOutgoingWGInterface(dst string) string {

	routes, err := GetWGRouteTable()
	if err != nil {
		return ""
	}

	for _, route := range routes {
		ipAddr := strings.Split(route.Destination.String(), ".")
		if ipAddr[1] == "201" && ipAddr[2] == dst && ipAddr[3] == "0" {
			return route.Iface
		}
	}

	return ""
}

// GetWGRouteTable reads the /proc/net/route file and returns a slice of Route objects, each
// representing a route in the table that is related to a WireGuard interface.
// Ignores any lines that do not correspond to a WireGuard interface.
func GetWGRouteTable() ([]Route, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var routes []Route
	scanner := bufio.NewScanner(file)

	// Skip the header line
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		if len(fields[0]) < 2 || fields[0][:2] != "wg" {
			continue
		}

		dest, err := parseIP(fields[1])
		if err != nil {
			continue
		}

		gateway, err := parseIP(fields[2])
		if err != nil {
			continue
		}

		flags, err := strconv.ParseInt(fields[3], 16, 32)
		if err != nil {
			continue
		}

		mask, err := parseIP(fields[7])
		if err != nil {
			continue
		}

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       int(flags),
			Iface:       fields[0],
			Mask:        net.IPMask(mask),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func parseIP(hexIP string) (net.IP, error) {
	rawIP, err := strconv.ParseUint(hexIP, 16, 32)
	if err != nil {
		return nil, err
	}
	ip := make(net.IP, 4)
	ip[0] = byte(rawIP & 0xFF)
	ip[1] = byte((rawIP >> 8) & 0xFF)
	ip[2] = byte((rawIP >> 16) & 0xFF)
	ip[3] = byte((rawIP >> 24) & 0xFF)
	return ip, nil
}
