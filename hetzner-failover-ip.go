package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/dmotylev/goproperties"
	"github.com/dmotylev/hetzner/api"
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
		log.Fatalf("got error: %s", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ip\tactive_server_ip\tserver_ip\tserver_number")
	for _, d := range ips {
		a := net.ParseIP(d.Netmask)
		_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
		fmt.Fprintf(w, "%s/%d\t%s\t%s\t%d\n", d.Ip, bits, d.Active_server_ip, d.Server_ip, d.Server_number)
	}
	w.Flush()
}

func printFailoverIp(addr string) {
	var failover api.Failover
	if err := api.Get("/failover/"+addr, &failover); err != nil {
		log.Fatalf("got error: %s", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ip\tactive_server_ip\tserver_ip\tserver_number")
	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Fprintf(w, "%s/%d\t%s\t%s\t%d\n", failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number)
	w.Flush()
}

func updateFailoverIp(addr, activeServerIp string) {
	var (
		failover api.Failover
		params   = url.Values{}
	)
	params.Add("active_server_ip", activeServerIp)
	if err := api.Post("/failover/"+addr, params, &failover); err != nil {
		log.Fatalf("got error: %s", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ip\tactive_server_ip\tserver_ip\tserver_number")
	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Fprintf(w, "%s/%d\t%s\t%s\t%d\n", failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number)
	w.Flush()
}

func main() {
	var (
		failoverIp = flag.String("failover-ip", "", "failover ip address")
		serverIp   = flag.String("active-server-ip", "", "new active server ip")
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
	case flag.NFlag() == 0:
		printAllFailoverIPs()
	case len(*failoverIp) != 0 && len(*serverIp) == 0:
		printFailoverIp(*failoverIp)
	case len(*failoverIp) != 0 && len(*serverIp) != 0:
		updateFailoverIp(*failoverIp, *serverIp)
	default:
		flag.Usage()
	}
}
