package common

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

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
		if p == "" {
			// rewrite as switch
			switch u.Scheme {
			case "http":
				port = 80
			case "https":
				port = 443
			case "ftp":
				port = 21
			case "ssh":
				port = 22
			case "ldap":
				port = 389
			case "ldaps":
				port = 636
			default:
				return host, 0, fmt.Errorf("unhandled url scheme %s", u.Scheme)
			}
		} else {
			port, err = strconv.Atoi(p)
		}
		return
	}

	h, p, e := net.SplitHostPort(input)
	if e == nil {
		host = h
		port, e = strconv.Atoi(p)
	}
	err = e
	return
}

// SetHostPort returns host:port from a string host and port int
func SetHostPort(host string, port int) string {
	if port == 0 {
		return host
	}
	p := fmt.Sprintf("%d", port)
	return net.JoinHostPort(host, p)
}
