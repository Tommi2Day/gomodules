// Package netlib provides a simple IP and DNS functions
package netlib

import (
	"context"
	"net"
	"time"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

type DNSconfig struct {
	Nameserver string
	Port       int
	TCP        bool
	Resolver   *net.Resolver
	Timeout    time.Duration
	IPv4Only   bool
	IPv6Only   bool
}

const defaultDNSTimeout = 5 * time.Second

// NewResolver returns a DNSconfig object
func NewResolver(nameserver string, port int, tcp bool) (dns *DNSconfig) {
	// network type
	nsIP := ""
	n := "udp"
	if tcp {
		n = "tcp"
	}
	ips, err := net.LookupHost(nameserver)
	if err != nil || len(ips) == 0 {
		log.Debugf("DNS lookup of %s failed: %s", nameserver, err)
	} else {
		nsIP = ips[0]
	}
	dns = new(DNSconfig)
	var resolver *net.Resolver
	if nsIP != "" {
		if port == 0 {
			port = 53
		}

		a := common.SetHostPort(nsIP, port)
		log.Debugf("Configured custom DNS resolver: %s", a)
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: defaultDNSTimeout,
				}
				return d.DialContext(ctx, n, a)
			},
		}
	} else {
		resolver = net.DefaultResolver
		log.Debug("use default DNS resolver")
	}
	dns.TCP = tcp
	dns.Port = port
	dns.Nameserver = nsIP
	dns.Resolver = resolver
	dns.Timeout = defaultDNSTimeout
	dns.IPv4Only = false
	dns.IPv6Only = false
	return
}

// LookupSrv looks up the SRV record of a service
func (dns *DNSconfig) LookupSrv(srv string, domain string) (srvEntries []*net.SRV, err error) {
	if dns.Resolver == nil {
		dns.Resolver = &net.Resolver{}
		dns.Timeout = defaultDNSTimeout
	}
	// add timeout
	ctx, cancel := context.WithTimeout(context.Background(), dns.Timeout)
	defer cancel()

	// create SRV record query
	_, srvEntries, err = dns.Resolver.LookupSRV(ctx, srv, "tcp", domain)
	if err != nil {
		log.Warnf("DNS: cannot resolve SRV with %s:%s", srv, err)
	}
	return
}

// LookupIP looks up the IP address of a hostname
func (dns *DNSconfig) LookupIP(hostname string) (ipEntires []net.IP, err error) {
	// is ip
	if IsValidIP(hostname) {
		ipEntires = append(ipEntires, net.ParseIP(hostname))
		return
	}
	if dns.Resolver == nil {
		dns.Resolver = &net.Resolver{}
		dns.Timeout = defaultDNSTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), dns.Timeout)
	defer cancel()

	// lookup type network
	n := "ip" // default ipv4+ipv6
	if dns.IPv4Only {
		n = "ip4"
		dns.IPv6Only = false
	}
	if dns.IPv6Only {
		n = "ip6"
	}

	ipEntires, err = dns.Resolver.LookupIP(ctx, n, hostname)
	if err != nil {
		log.Warnf("DNS: cannot resolve Host %s:%s", hostname, err)
	}
	return
}

// LookupTXT looks up the TXT record of a hostname
func (dns *DNSconfig) LookupTXT(hostname string) (txt []string, err error) {
	if dns.Resolver == nil {
		dns.Resolver = &net.Resolver{}
		dns.Timeout = defaultDNSTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), dns.Timeout)
	defer cancel()
	txt, err = dns.Resolver.LookupTXT(ctx, hostname)
	if err != nil {
		log.Warnf("DNS: cannot resolve TXT for %s:%s", hostname, err)
		return
	}
	return
}

// IsValidIP checks if the input string is a valid IP address
func IsValidIP(ip string) bool {
	i := net.ParseIP(ip)
	return i != nil
}

// IsPrivateIP checks if the input string is a private IP address
func IsPrivateIP(ip string) bool {
	i := net.ParseIP(ip)
	if i == nil {
		return false
	}
	return i.IsPrivate()
}

// IsIPv4 checks if the input string is a validIPv4 address
func IsIPv4(ip string) bool {
	i := net.ParseIP(ip)
	if i == nil {
		return false
	}
	ip4 := i.To4()
	return ip4 != nil
}

// IsIPv6 checks if the input string is a valid IPv6 address
func IsIPv6(ip string) bool {
	return IsValidIP(ip) && !IsIPv4(ip)
}
