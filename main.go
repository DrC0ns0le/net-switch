package main

import (
	"github.com/DrC0ns0le/net-switch/link"
)

var (
	prometheusAddr = "http://10.1.1.109:8428"
)

func main() {
	link.Init(prometheusAddr)
}
