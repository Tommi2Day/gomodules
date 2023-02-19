// Package ldaplib collects ldap related functions
package ldaplib

import (
	"crypto/tls"
	"fmt"

	ldap "github.com/go-ldap/ldap/v3"
)

// ConfigType helds config properties
type ConfigType struct {
	Server   string
	Port     int
	URL      string
	TLS      bool
	Insecure bool
}

var ldapConfig ConfigType

// SetConfig defines common connection parameter
func SetConfig(server string, port int, tls bool, insecure bool) {
	if port == 0 {
		if tls {
			port = 636
		} else {
			port = 389
		}
	}
	ldapConfig.Server = server
	ldapConfig.Port = port
	ldapConfig.TLS = tls
	ldapConfig.Insecure = insecure
	ldapConfig.URL = fmt.Sprintf("ldap://%s:%d", ldapConfig.Server, ldapConfig.Port)
	if tls {
		ldapConfig.URL = fmt.Sprintf("ldaps://%s:%d", ldapConfig.Server, ldapConfig.Port)
	}
}

// GetConfig retrievs actual config
func GetConfig() ConfigType {
	return ldapConfig
}

// Connect will authorize to the ldap server
func Connect(bindDN string, bindPassword string) (l *ldap.Conn, err error) {
	// You can also use IP instead of FQDN
	if ldapConfig.Insecure {
		//nolint gosec
		l, err = ldap.DialURL(ldapConfig.URL, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	} else {
		l, err = ldap.DialURL(ldapConfig.URL)
	}

	if err != nil {
		return nil, err
	}
	if len(bindDN) == 0 {
		err = l.UnauthenticatedBind("")
	} else {
		err = l.Bind(bindDN, bindPassword)
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Search do a search on ldap
func Search(l *ldap.Conn, baseDN string, filter string, attributes []string, scope int, deref int) (*ldap.SearchResult, error) {
	searchReq := ldap.NewSearchRequest(
		baseDN,
		scope, // https://pkg.go.dev/github.com/go-ldap/ldap/v3@v3.4.4#ScopeWholeSubtree
		deref, //https://pkg.go.dev/github.com/go-ldap/ldap/v3@v3.4.4#DerefInSearching
		0,
		0,
		false,
		filter,
		attributes,
		nil,
	)
	result, err := l.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search Error: %s", err)
	}

	if len(result.Entries) > 0 {
		return result, nil
	}
	return nil, fmt.Errorf("no entries found")
}
