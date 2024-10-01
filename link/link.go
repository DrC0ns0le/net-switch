package link

import (
	"log"
	"math"
	"os/exec"
	"time"

	perfutils "github.com/DrC0ns0le/net-perf/utils"
	"github.com/DrC0ns0le/net-switch/utils"
)

var (
	localID        string
	prometheusAddr string

	justStarted = true
)

func Init(addr string) {
	id, err := perfutils.GetLocalID()
	if err != nil {
		panic(err)
	}

	localID = id
	prometheusAddr = addr

	startWorker()

}

func startWorker() {
	log.Println("starting worker")
	mustUpdateRoutes()
	justStarted = false
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			go mustUpdateRoutes()
		}
	}
}

func mustUpdateRoutes() {

	ifaces, err := perfutils.GetAllWGInterfaces()
	if err != nil {
		log.Panicf("failed to get interfaces: %v", err)
	}
	utils.EnableAsymmetricRoute()

	var remoteVersionMap = map[string][]string{}

	for _, iface := range ifaces {
		remoteVersionMap[iface.RemoteID] = append(remoteVersionMap[iface.RemoteID], iface.IPVersion)
	}

	for remote, versions := range remoteVersionMap {
		var version string
		var reason string
		if len(versions) > 1 {
			version, reason, err = choosePreferredVersion(remote)
			if err != nil {
				log.Printf("failed to determine preferred version: %v", err)
			}

			if justStarted {
				version = "4"
			} else if version == "" {
				// log.Printf("neither v4 nor v6 are preferred for %s, skipping", remote)
				continue
			}
		} else {
			version = versions[0]
		}
		// check which interface is being used
		iface := utils.GetOutgoingWGInterface(remote)

		if iface == "" {
			log.Printf("route for %s not found in the routing table, adding route via %s", remote, "wg"+localID+"."+remote+"_v"+version)
			// run "ip route add 10.201.{remote}.0/24 dev wg{local}.{remote}_v{version} scope link src 10.201.{local}.1"
			err := exec.Command("ip", "route", "add", "10.201."+remote+".0/24", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link", "src", "10.201."+localID+".1").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
			// run "ip -6 route add fdac:c9:{remote}::/64 dev wg{local}.{remote}_v{version} scope link"
			err = exec.Command("ip", "-6", "route", "add", "fdac:c9:"+remote+"::/64", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
		} else if iface != "wg"+localID+"."+remote+"_v"+version {
			// run "ip route change 10.201.{remote}.0/24 dev wg{local}.{remote}_v{version} scope link"
			log.Printf("changing route for %s from %s to %s due to %s", remote, iface, "wg"+localID+"."+remote+"_v"+version, reason)
			err := exec.Command("ip", "route", "change", "10.201."+remote+".0/24", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link", "src", "10.201."+localID+".1").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
			// run "ip -6 route change fdac:c9:{remote}::/64 dev wg{local}.{remote}_v{version} scope link"
			err = exec.Command("ip", "-6", "route", "change", "fdac:c9:"+remote+"::/64", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
		} else {
			// log.Printf("route for %s already set to preferred interface %s", remote, iface)
		}
	}
}

// choosePreferredVersion takes a remote endpoint string and returns the preferred
// IP version to use for communication with that endpoint. It does this by
// comparing the latency and packet loss metrics for both IPv4 and IPv6
// connections to the remote endpoint. If the metrics are equal, it returns an
// empty string. If the metrics for one version are significantly better than
// the other, it returns that version. Otherwise, it returns the version with the
// lowest latency.
func choosePreferredVersion(remote string) (string, string, error) {

	metrics, err := getMetrics(remote)
	if err != nil {
		return "", "", err
	}

	versionScoreMap := map[string]float64{
		"4": 0.0,
		"6": 0.0,
	}

	for v, m := range metrics {
		if m.Latency == 0 {
			continue
		}
		versionScoreMap[v] = 1 / ((m.Latency / 1e6) * (math.Sqrt(m.PacketLoss / 100)))
	}

	if int(metrics["4"].Availability) != 1 && int(metrics["6"].Availability) != 1 {
		if metrics["4"].Availability > metrics["6"].Availability {
			return "4", "higher availability", nil
		} else {
			return "6", "higher availability", nil
		}
	} else if versionScoreMap["4"] == math.Inf(1) && versionScoreMap["6"] == math.Inf(1) {
		if metrics["4"].Latency < metrics["6"].Latency {
			return "4", "lower latency", nil
		} else {
			return "6", "lower latency", nil
		}
	} else {
		if versionScoreMap["4"] > versionScoreMap["6"] {
			return "4", "higher score", nil
		} else {
			return "6", "higher score", nil
		}
	}
}
