package dblib

import (
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/netlib"
	"gopkg.in/ini.v1"
)

// ServiceEntryType holds  host/ip/port of a tns address section
type ServiceEntryType struct {
	Host    string
	IP      string
	Port    string
	Address string
}

// ServiceEntries List of Map of service entries
type ServiceEntries []ServiceEntryType

// IgnoreDNSLookup if true, no oracle-dns lookup is done
var IgnoreDNSLookup = false

// IPv4Only if true, only IPv4 addresses are returned
var IPv4Only = true

// DNSConfig holds the netlib DNS object
var DNSConfig *netlib.DNSconfig

// GetRacAdresses reads racinfo.ini or DNS SRV and returns all IP addresses for given rac
func GetRacAdresses(rachost string, racini string) (services ServiceEntries) {
	if racini != "" {
		services = getRACAddressesFromRacInfo(rachost, racini)
		log.Debugf("add %d rac addresses for %s from file %s", len(services), rachost, racini)
	}
	if len(services) == 0 {
		services = getRACAddressesFromDNSSrv(rachost)
		log.Debugf("add %d rac addresses for %s from DNS SRV records", len(services), rachost)
	}
	return
}

// getRACAddressesFromDNSSrv returns a list of RAC IP addresses for given tnshost using DNS SRV lookup
// SRV Entry format for rachost=rac.example.com:
// _rac._tcp.example.com 10 5 80 racscan.example.com.
// _rac._tcp.example.com 10 5 80 rac-vip1.example.com.
// _rac._tcp.example.com 10 5 80 rac-vip2.example.com.
func getRACAddressesFromDNSSrv(rachost string) (services ServiceEntries) {
	if IgnoreDNSLookup {
		log.Infof("DNSSrv: Skip SRV, Ignore DNS is set")
		return
	}
	// check if host contains only digits
	if netlib.IsValidIP(rachost) {
		log.Debugf("DNSSrv: %s is an IP address, skip", rachost)
		return
	}

	// configure resolver
	if DNSConfig == nil {
		// set default resolver
		DNSConfig = netlib.NewResolver("", 0, false)
	}
	// split host and domain
	domain := ""
	parts := strings.Split(rachost, ".")
	host := parts[0]
	if len(parts) > 1 {
		domain = strings.Join(parts[1:], ".")
	}
	// create SRV record query
	srvList, err := DNSConfig.LookupSrv(host, domain)
	if err != nil {
		log.Warnf("DNSSrv: cannot resolve %s:%s", host, err)
		return
	}

	// process returned addresses
	for _, srv := range srvList {
		host = srv.Target
		// delete trailing dot
		host = strings.TrimSuffix(host, ".")
		port := srv.Port
		services = append(services, getServiceList(host, fmt.Sprintf("%v", port))...)
	}
	log.Debugf("DNSSrv: Rac %s Add %d adresses", rachost, len(services))
	return
}

// getRACAddressesFromRacInfo returns a list of RAC IP addresses from inifile (default: racinfo.ini)
func getRACAddressesFromRacInfo(rachost string, filename string) (services ServiceEntries) {
	cfg, err := ini.InsensitiveLoad(filename)
	if err != nil {
		log.Debugf("RacInfo: cannot Read %s:%s", filename, err)
		return
	}
	// all keys are lowwer case
	entries := cfg.Section(strings.ToLower(rachost)).Keys()
	if len(entries) == 0 {
		log.Debugf("RacInfo: no entries for %s in %s", rachost, filename)
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(strings.ToLower(e.Name()), "vip") || strings.HasPrefix(strings.ToLower(e.Name()), "scan") {
			a := e.Value()
			host, port, err := net.SplitHostPort(a)
			if err != nil {
				log.Warnf("RacInfo: cannot parse %s:%s", a, err)
				continue
			}
			services = append(services, getServiceList(host, fmt.Sprintf("%v", port))...)
		}
	}
	log.Debugf("RacInfo: Rac %s Add %d adresses", rachost, len(services))
	return
}

// getServiceList returns a list of IP addresses for given tnshost and port
func getServiceList(host string, port string) (services ServiceEntries) {
	// configure resolver
	// configure resolver
	if DNSConfig == nil {
		// set default resolver
		DNSConfig = netlib.NewResolver("", 0, false)
	}

	// set resolver network ip =ipv4+ipv6 or ip4 only
	if IPv4Only {
		DNSConfig.IPv4Only = true
	}

	ips, err := DNSConfig.LookupIP(host)
	if err != nil || len(ips) == 0 {
		if IgnoreDNSLookup {
			log.Debugf("getServiceList: cannot resolve %s", host)
			service := ServiceEntryType{Host: host, Port: port, IP: "", Address: fmt.Sprintf("%s:%v", host, port)}
			services = append(services, service)
			return
		}
		log.Warnf("cannot resolve %s, skipped", host)
		return
	}
	for _, ip := range ips {
		hostip := ip.String()
		if IPv4Only && !netlib.IsIPv4(hostip) {
			log.Debugf("getServiceList: skip non ipv4 %s", hostip)
			continue
		}
		service := ServiceEntryType{Host: host, Port: port}
		if len(hostip) > 0 {
			service.IP = hostip
			service.Address = fmt.Sprintf("%s:%v", hostip, port)
		} else {
			service.IP = ""
			service.Address = fmt.Sprintf("%s:%v", host, port)
		}
		services = append(services, service)
	}
	log.Debugf("add %d Services for %s:%s", len(services), host, port)
	return
}
