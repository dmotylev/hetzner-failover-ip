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
	format      = "%s\t/%d\t%s\t%s\t%d\t%c\n"
	markOnDuty  = '+'
	markStandBy = '-'
)

func unbrace(s string) string {
	if len(s) > 1 && (s[0] == '\'' || s[0] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}

func fatal(err error) {
	log.Println(err)
	if e, ok := err.(*api.RequestError); ok {
		os.Exit(e.HttpStatusCode - 300) // downgrade http status code from expected range ]400-500[ to byte size
	}
	os.Exit(1)
}

func dutyMark(ip1, ip2 string) rune {
	if ip1 == ip2 {
		return markOnDuty
	}
	return markStandBy
}

func printAllFailoverIPs(failoverIp, localIp string) {
	var ips []api.Failover
	if err := api.Get("/failover", &ips); err != nil {
		fatal(err)
	}

	for _, d := range ips {
		a := net.ParseIP(d.Netmask)
		_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
		mark := ' '
		if d.Ip.String() == failoverIp {
			mark = dutyMark(d.Active_server_ip.String(), localIp)
		}
		fmt.Printf(format, d.Ip, bits, d.Active_server_ip, d.Server_ip, d.Server_number, mark)
	}
}

func printFailoverIp(failoverIp, localIp string) {
	var failover api.Failover
	if err := api.Get("/failover/"+failoverIp, &failover); err != nil {
		fatal(err)
	}

	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Printf(format, failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number, dutyMark(failover.Active_server_ip.String(), localIp))
}

func checkDutyStatus(failoverIp, localIp string) {
	var failover api.Failover
	if err := api.Get("/failover/"+failoverIp, &failover); err != nil {
		fatal(err)
	}
	if failover.Active_server_ip.String() == localIp {
		os.Exit(0)
	}
	os.Exit(255)
}

func updateFailoverIp(failoverIp, activeServerIp, localIp string) {
	var (
		failover api.Failover
		params   = url.Values{}
	)
	params.Add("active_server_ip", activeServerIp)
	if err := api.Post("/failover/"+failoverIp, params, &failover); err != nil {
		fatal(err)
	}

	a := net.ParseIP(failover.Netmask)
	_, bits := net.IPv4Mask(a[0], a[1], a[2], a[3]).Size()
	fmt.Printf(format, failover.Ip, bits, failover.Active_server_ip, failover.Server_ip, failover.Server_number, dutyMark(failover.Active_server_ip.String(), localIp))
}

func main() {
	log.SetFlags(0)

	// load credentials
	rc, err := properties.Load(os.ExpandEnv("$HOME/.hetzner.rc"))
	if err != nil {
		rc, err = properties.Load(os.ExpandEnv("/etc/hetzner-api.conf"))
	}

	if err != nil {
		log.Fatalf("no credentials: %s", err)
	}

	var (
		failoverIp         = flag.String("f", rc["failover-ip"], "Failover ip address (default to 'failover-ip' value in rc)")
		localIp            = flag.String("l", rc["local-ip"], "ip address of this server (default to 'local-ip' value in rc)")
		serverIp           = flag.String("s", "", "new active Server ip")
		allFailoversWanted = flag.Bool("a", false, "list All failover ips (failover ip, mask, active server ip, server ip, server number)")
		dutyStatusWanted   = flag.Bool("t", false, "Test if local server is the active; returns 0 for active 255 otherwise")
		takeWanted         = flag.Bool("take", false, "set local ip as new active server ip")
	)
	flag.Parse()

	api.SetBasicAuth(unbrace(rc["login"]), unbrace(rc["password"]))

	switch {
	case *takeWanted:
		updateFailoverIp(*failoverIp, *localIp, *localIp)
	case *dutyStatusWanted:
		checkDutyStatus(*failoverIp, *localIp)
	case *allFailoversWanted:
		printAllFailoverIPs(*failoverIp, *localIp)
	case len(*failoverIp) != 0 && len(*serverIp) == 0:
		printFailoverIp(*failoverIp, *localIp)
	case len(*failoverIp) != 0 && len(*serverIp) != 0:
		updateFailoverIp(*failoverIp, *serverIp, *localIp)
	default:
		flag.Usage()
	}
}
