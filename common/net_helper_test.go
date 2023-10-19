package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostPort(t *testing.T) {
	t.Run("Check Host parsing", func(t *testing.T) {
		type testTableType struct {
			name    string
			input   string
			success bool
			host    string
			port    int
		}
		for _, testconfig := range []testTableType{
			{
				name:    "only host",
				input:   "localhost",
				success: false,
				host:    "localhost",
				port:    0,
			},
			{
				name:    "host and port",
				input:   "localhost:1234",
				success: true,
				host:    "localhost",
				port:    1234,
			},
			{
				name:    "with tcp url",
				input:   "tcp://docker:2375",
				success: true,
				host:    "docker",
				port:    2375,
			},
			{
				name:    "with http url",
				input:   "http://localhost:8080/app/index.html",
				success: true,
				host:    "localhost",
				port:    8080,
			},
			{
				name:    "with http url without port",
				input:   "http://localhost/app/index.html",
				success: true,
				host:    "localhost",
				port:    80,
			},
			{
				name:    "with ssh url without port",
				input:   "ssh://localhost",
				success: true,
				host:    "localhost",
				port:    22,
			},
			{
				name:    "with ldap url and user/password",
				input:   "ldap://user:password@ldapserver.de",
				success: true,
				host:    "ldapserver.de",
				port:    389,
			},
			{
				name:    "with ldap url without port",
				input:   "ldaps://ldapserver",
				success: true,
				host:    "ldapserver",
				port:    636,
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				host, port, err := GetHostPort(testconfig.input)
				if testconfig.success {
					assert.NoErrorf(t, err, "unexpected error %s", err)
					assert.Equalf(t, testconfig.host, host, "entry returned wrong host ('%s' <>'%s)", host, testconfig.host)
					assert.Equalf(t, testconfig.port, port, "entry returned wrong port ('%d' <>'%d)", port, testconfig.port)
				} else {
					assert.Error(t, err, "Expected error not set")
				}
			})
		}
	})
}
func TestSetHostPort(t *testing.T) {
	t.Run("Test SetHostPort ipv4", func(t *testing.T) {
		actual := SetHostPort("localhost", 1234)
		assert.Equalf(t, "localhost:1234", actual, "actual not expected %s", actual)
	})
	t.Run("Test SetHostPort tcpv6", func(t *testing.T) {
		actual := SetHostPort("fe80::3436:bd7c:3037:df6f", 1234)
		assert.Equalf(t, "[fe80::3436:bd7c:3037:df6f]:1234", actual, "actual not expected: %s", actual)
	})
	t.Run("Test SetHostPort noport", func(t *testing.T) {
		actual := SetHostPort("localhost", 0)
		assert.Equalf(t, "localhost", actual, "actual not expected: %s", actual)
	})
}
