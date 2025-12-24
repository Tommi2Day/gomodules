package netlib

import (
	"fmt"
	"os"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/stretchr/testify/assert"
)

const (
	tLdap    = "ldap." + netlibDomain
	tLdapIP4 = "ldap-ip4." + netlibDomain
	tDB      = "db." + netlibDomain
)

func TestMain(m *testing.M) {
	var err error

	test.InitTestDirs()
	if os.Getenv("SKIP_NET_DNS") != "" {
		fmt.Println("Skipping Net DNS Container in CI environment")
		return
	}

	netlibDNSContainer, err = prepareNetlibDNSContainer()
	if err != nil || netlibDNSContainer == nil {
		// run even non docker test if prepare failed
		_ = os.Setenv("SKIP_NET_DNS", "true")
		fmt.Printf("prepareNetlibDNSContainer failed: %s", err)
	}
	code := m.Run()
	destroyDNSContainer(netlibDNSContainer)
	os.Exit(code)
}

func TestIPs(t *testing.T) {
	t.Run("Test IsValidIP", func(t *testing.T) {
		type testTableType struct {
			name     string
			input    string
			expected bool
		}
		for _, testconfig := range []testTableType{
			{
				name:     "ipv4",
				input:    "127.0.0.1",
				expected: true,
			},
			{
				name:     "ipv6",
				input:    "fe80::3436:bd7c:3037:df6f",
				expected: true,
			},
			{
				name:     "hostname",
				input:    "localhost",
				expected: false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				actual := IsValidIP(testconfig.input)
				assert.Equalf(t, testconfig.expected, actual, "actual not expected %t", actual)
			})
		}
	})

	t.Run("Test IsIPv4", func(t *testing.T) {
		type testTableType struct {
			name     string
			input    string
			expected bool
		}
		for _, testconfig := range []testTableType{
			{
				name:     "private ipv4 192.168",
				input:    "192.168.0.1",
				expected: true,
			},
			{
				name:     "public ipv4",
				input:    "100.20.2.20",
				expected: true,
			},
			{
				name:     "public ipv4 172.10",
				input:    "172.10.0.1",
				expected: true,
			},
			{
				name:     "ipv4 short",
				input:    "172.17.0",
				expected: false,
			},
			{
				name:     "private ipv6",
				input:    "fd00::1",
				expected: false,
			},
			{
				name:     "public ipv6",
				input:    "fe80::3436:bd7c:3037:df6f",
				expected: false,
			},
			{
				name:     "invalid",
				input:    "invalid.host",
				expected: false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				actual := IsIPv4(testconfig.input)
				assert.Equalf(t, testconfig.expected, actual, "actual not expected %t", actual)
			})
		}
	})

	t.Run("Test IsIPv6", func(t *testing.T) {
		type testTableType struct {
			name     string
			input    string
			expected bool
		}
		for _, testconfig := range []testTableType{

			{
				name:     "private ipv4 12.0.0.1",
				input:    "172.10.0.1",
				expected: false,
			},
			{
				name:     "public ipv4 172.10",
				input:    "172.10.0.1",
				expected: false,
			},
			{
				name:     "private ipv6",
				input:    "fd00::1",
				expected: true,
			},
			{
				name:     "public ipv6",
				input:    "fe80::3436:bd7c:3037:df6f",
				expected: true,
			},
			{
				name:     "invalid",
				input:    "invalid.host",
				expected: false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				actual := IsIPv6(testconfig.input)
				assert.Equalf(t, testconfig.expected, actual, "actual not expected %t", actual)
			})
		}
	})
	t.Run("Test IsPrivate", func(t *testing.T) {
		type testTableType struct {
			name     string
			input    string
			expected bool
		}
		for _, testconfig := range []testTableType{
			{
				name:     "private ipv4 192.168",
				input:    "192.168.0.1",
				expected: true,
			},
			{
				name:     "private ipv4 172.16",
				input:    "172.18.0.1",
				expected: true,
			},
			{
				name:     "private ipv4 10",
				input:    "10.200.0.1",
				expected: true,
			},
			{
				name:     "public ipv4",
				input:    "100.20.2.20",
				expected: false,
			},
			{
				name:     "public ipv4 172.10",
				input:    "172.10.0.1",
				expected: false,
			},
			{
				name:     "private ipv6",
				input:    "fd00::1",
				expected: true,
			},
			{
				name:     "public ipv6",
				input:    "fe80::3436:bd7c:3037:df6f",
				expected: false,
			},
			{
				name:     "invalid",
				input:    "invalid.host",
				expected: false,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				actual := IsPrivateIP(testconfig.input)
				assert.Equalf(t, testconfig.expected, actual, "actual not expected %t", actual)
			})
		}
	})
}

func TestDNSConfig(t *testing.T) {
	t.Run("Test default Resolver", func(t *testing.T) {
		dns := NewResolver("", 0, false)
		assert.NotNil(t, dns.Resolver, "Resolver not set")
		assert.False(t, dns.TCP, "TCP set, but not expected")
		assert.Equal(t, defaultDNSTimeout, dns.Timeout, "Timeout not expected")
	})
	t.Run("Test private Resolver", func(t *testing.T) {
		dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
		// assert.Equal(t, netlibDNSServer, dns.Nameserver, "Server not expected")
		assert.Equal(t, netlibDNSPort, dns.Port, "Port not expected")
		assert.NotNil(t, dns.Resolver, "Resolver not set")
		assert.True(t, dns.TCP, "TCP not set")
		assert.Equal(t, defaultDNSTimeout, dns.Timeout, "Timeout not expected")
		t.Logf("Resolver: %s:%d(TCP:%v)\n", dns.Nameserver, dns.Port, dns.TCP)
	})
	t.Run("Test unset Resolver", func(t *testing.T) {
		if os.Getenv("SKIP_PUBLIC_DNS") != "" {
			t.Skip("Skipping public DNS testing in CI environment")
		}
		dns := new(DNSconfig)
		ips, err := dns.LookupIP("www.google.com")
		assert.NoError(t, err, "unexpected error")
		assert.Greater(t, len(ips), 0, "entry empty")
		t.Logf("IPs: %v", ips)
	})
}
func TestLookupSrv(t *testing.T) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		t.Skip("Skipping Net DNS testing")
	}
	// use DNS from Docker
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	actual, e := dns.LookupSrv("ldap", netlibDomain)
	assert.NoError(t, e, "unexpected error")
	assert.NotEmpty(t, actual, "entry empty")
	l := len(actual)
	assert.Greater(t, l, 0, "entry empty")
	if l > 0 {
		h := actual[0].Target
		p := int(actual[0].Port)
		assert.Equal(t, tLdap+".", h, "Target %s not expected", h)
		assert.Equalf(t, 389, p, "Port %d not expected", p)
		t.Logf("SRV: Target: %s, Port: %d", h, p)
	}
}

func TestLookupIP(t *testing.T) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		t.Skip("Skipping NET DNS testing")
	}
	// use DNS from Docker
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	type testTableType struct {
		name     string
		input    string
		expected bool
		iptype   string
		l        int
	}

	for _, testconfig := range []testTableType{
		{
			name:     "ipv4",
			input:    "127.0.0.1",
			expected: true,
			iptype:   "ipv4",
			l:        1,
		},
		{
			name:     "ipv6",
			input:    "fe80::3436:bd7c:3037:df6f",
			expected: true,
			iptype:   "ipv6",
			l:        1,
		},
		{
			name:     "hostname both",
			input:    tLdap,
			expected: true,
			iptype:   "",
			l:        2,
		},
		{
			name:     "hostname ipv4",
			input:    tLdap,
			expected: true,
			iptype:   "ipv4",
			l:        1,
		},
		{
			name:     "hostname ipv6",
			input:    tLdap,
			expected: true,
			iptype:   "ipv6",
			l:        1,
		},
		{
			name:     "hostname ipv4",
			input:    tLdapIP4,
			expected: true,
			iptype:   "",
			l:        1,
		},
		{
			name:     "hostname ipv4 query ipv6",
			input:    tLdapIP4,
			expected: false,
			iptype:   "ipv6",
			l:        0,
		},
		{
			name:     "invalid",
			input:    "invalid.host",
			expected: false,
		},
	} {
		t.Run(testconfig.name, func(t *testing.T) {
			dns.IPv4Only = false
			dns.IPv6Only = false

			if testconfig.iptype == "ipv4" {
				dns.IPv4Only = true
			}
			if testconfig.iptype == "ipv6" {
				dns.IPv6Only = true
			}

			actual, e := dns.LookupIP(testconfig.input)
			if testconfig.expected {
				assert.NoErrorf(t, e, "unexpected error %s", e)
				assert.NotEmptyf(t, actual, "entry empty")
				assert.Equalf(t, testconfig.l, len(actual), "entry size not expected")
			} else {
				assert.Error(t, e, "Expected error not set")
			}
		})
	}
}

func TestLookupIPV4V6(t *testing.T) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		t.Skip("Skipping NET_DNS testing")
	}
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)
	dns.IPv4Only = false
	dns.IPv6Only = false
	actual, e := dns.LookupIP(tDB)
	assert.NoError(t, e, "unexpected error")
	assert.NotEmpty(t, actual, "entry empty")
	l := len(actual)
	assert.Greater(t, l, 0, "entry empty")
	ipv4 := false
	ipv6 := false
	if l > 0 {
		for v := range actual {
			ip := actual[v].String()
			t.Logf("IP: %s", ip)
			if IsIPv4(ip) {
				ipv4 = true
			}
			if IsIPv6(ip) {
				ipv6 = true
			}
		}
		assert.True(t, ipv4, "No ipv4 address found")
		assert.True(t, ipv6, "No ipv6 address found")
	}
}
func TestLookupTXT(t *testing.T) {
	if os.Getenv("SKIP_NET_DNS") != "" {
		t.Skip("Skipping NET_DNS testing")
	}

	// use DNS from Docker
	dns := NewResolver(netlibDNSServer, netlibDNSPort, true)

	actual, e := dns.LookupTXT(tDB)
	assert.NoError(t, e, "unexpected error")
	assert.NotEmpty(t, actual, "entry empty")
	l := len(actual)
	assert.Greater(t, l, 0, "entry empty")
	if l > 0 {
		txt := actual[0]
		assert.Equal(t, "Database server", txt, "TXT %s not expected", txt)
	}
}
