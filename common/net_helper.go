package common

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const hostDefaultDomain = "localdomain"

var (
	// DefaultPorts is a map of default ports for common services
	DefaultPorts = map[string]int{
		"http":           80,
		"https":          443,
		"ftp":            21,
		"ftps":           990,
		"ssh":            22,
		"ldap":           389,
		"ldaps":          636,
		"imap":           143,
		"imaps":          993,
		"smtp":           25,
		"smtps":          465,
		"pop3":           110,
		"pop3s":          995,
		"mysql":          3306,
		"postgresql":     5432,
		"oracle":         1521,
		"mssql":          1433,
		"mongodb":        27017,
		"redis":          6379,
		"memcached":      11211,
		"couchdb":        5984,
		"couchbase":      8091,
		"cassandra":      9042,
		"elasticsearch":  9200,
		"kibana":         5601,
		"grafana":        3000,
		"prometheus":     9090,
		"alertmanager":   9093,
		"consul":         8500,
		"vault":          8200,
		"nomad":          4646,
		"etcd":           2379,
		"zookeeper":      2181,
		"kafka":          9092,
		"rabbitmq":       5672,
		"nats":           4222,
		"nats-streaming": 4223,
		"mqtt":           1883,
		"coap":           5683,
		"dns":            53,
		"dhcp":           67,
		"tftp":           69,
		"ntp":            123,
		"snmp":           161,
		"syslog":         514,
		"radius":         1812,
		"radius-acct":    1813,
	}
)

// HTTPGet returns the body of a GET request
func HTTPGet(url string, timeout int) (resp string, err error) {
	if timeout == 0 {
		timeout = 5
	}
	tr := new(http.Transport)
	client := &http.Client{Transport: tr}
	client.Timeout = time.Duration(timeout) * time.Second
	rc, err := client.Get(url)
	if err != nil {
		return
	}
	data, _ := io.ReadAll(rc.Body)
	return string(data), nil
}

// GetHostPort returns host and port from a string
func GetHostPort(input string) (host string, port int, err error) {
	if input == "" {
		return "", 0, fmt.Errorf("empty input")
	}
	var u *url.URL
	if strings.Contains(input, "://") {
		u, err = url.Parse(input)
		if err != nil {
			return "", 0, err
		}
		host = u.Hostname()
		p := u.Port()
		if p != "" {
			port, err = strconv.Atoi(p)
			return
		}
		// rewrite as switch
		dp, f := DefaultPorts[u.Scheme]
		if !f {
			return host, 0, fmt.Errorf("unhandled url scheme %s", u.Scheme)
		}
		port = dp
		return
	}

	if strings.LastIndex(input, ":") > 0 {
		h, p, e := net.SplitHostPort(input)
		if e == nil {
			host = h
			port, e = strconv.Atoi(p)
		}
		err = e
		return
	}
	return input, 0, nil
}

// SetHostPort returns host:port from a string host and port int
func SetHostPort(host string, port int) string {
	if port == 0 {
		return host
	}
	p := fmt.Sprintf("%d", port)
	return net.JoinHostPort(host, p)
}

// GetHostname returns the hostname as full qualified domain name
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost." + hostDefaultDomain
	}
	hostname = strings.ReplaceAll(hostname, "\\", ".")
	if !strings.Contains(hostname, ".") {
		hostname += "." + hostDefaultDomain
	}
	return hostname
}
