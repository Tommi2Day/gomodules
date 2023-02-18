// Package ldaplib collects ldap related functions
package ldaplib

import (
	"fmt"

	ldap "github.com/go-ldap/ldap/v3"
)

// ConfigType helds config properties
type ConfigType struct {
	Server       string
	Port         int
	TLS          bool
	BindDN       string
	BindPassword string
}

var ldapConfig ConfigType

// SetConfig defines common connection parameter
func SetConfig(server string, port int, tls bool) {
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
}

// GetConfig retrievs actual config
func GetConfig() ConfigType {
	return ldapConfig
}

// Connect will authorize to the ldap server
func Connect(bindDN string, bindPassword string) (*ldap.Conn, error) {
	// You can also use IP instead of FQDN
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:%d", ldapConfig.Server, ldapConfig.Port))
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
