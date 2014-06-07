package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/dmotylev/goproperties"
	"github.com/dmotylev/hetzner/api"
)

const (
	format = "%s\t/%d\t%s\t%s\t%d\n"
)

func unbrace(s string) string {
	if len(s) > 1 && (s[0] == '\'' || s[0] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}

func printAllFailoverIPs() {
	var ips []api.Failover
	if err := api.Get("/failover", &ips); err != nil {
		log.Fatal(err)
	}

	for _, d := range ips {
		a := net.ParseIP(d.Netmask)
		_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
		fmt.Printf(format, d.Ip, bits, d.Active_server_ip, d.Server_ip, d.Server_number)
	}
}

func printFailoverIp(addr string) {
	var failover api.Failover
	if err := api.Get("/failover/"+addr, &failover); err != nil {
		log.Fatal(err)
	}

	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Printf(format, failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number)
}

func updateFailoverIp(addr, activeServerIp string) {
	var (
		failover api.Failover
		params   = url.Values{}
	)
	params.Add("active_server_ip", activeServerIp)
	if err := api.Post("/failover/"+addr, params, &failover); err != nil {
		log.Fatal(err)
	}

	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Printf(format, failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number)
}

func main() {
	var (
		failoverIp         = flag.String("f", "", "Failover ip address")
		serverIp           = flag.String("s", "", "new active Server ip")
		allFailoversWanted = flag.Bool("l", false, "List all failover ips (failover ip, mask, active server ip, server ip, server number)")
	)
	flag.Parse()

	log.SetFlags(0)

	// load credentials
	rc, err := properties.Load(os.ExpandEnv("$HOME/.hetzner.rc"))
	if err != nil {
		rc, err = properties.Load(os.ExpandEnv("/etc/hetzner-api.conf"))
	}

	if err != nil {
		log.Fatalf("no credentials: %s", err)
	}

	api.SetBasicAuth(unbrace(rc["login"]), unbrace(rc["password"]))

	switch {
	case *allFailoversWanted:
		printAllFailoverIPs()
	case len(*failoverIp) != 0 && len(*serverIp) == 0:
		printFailoverIp(*failoverIp)
	case len(*failoverIp) != 0 && len(*serverIp) != 0:
		updateFailoverIp(*failoverIp, *serverIp)
	default:
		flag.Usage()
	}
}
