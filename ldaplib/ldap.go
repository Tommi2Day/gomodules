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
	BaseDN   string
}

var ldapConfig ConfigType

// SetConfig defines common connection parameter
func SetConfig(server string, port int, tls bool, insecure bool, basedn string) {
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
	ldapConfig.BaseDN = basedn
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
func Search(l *ldap.Conn, baseDN string, filter string, attributes []string, scope int, deref int) (entries []*ldap.Entry, err error) {
	var result *ldap.SearchResult
	if l == nil {
		err = fmt.Errorf("ldap delete no valid ldap handler")
		return
	}
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

	result, err = l.Search(searchReq)
	if err != nil {
		if ldap.IsErrorWithCode(err, 32) {
			return nil, nil
		}
		return nil, err
	}
	entries = result.Entries
	return
}

// DeleteEntry deletes given DN from Ldap
func DeleteEntry(l *ldap.Conn, dn string) (err error) {
	if l == nil {
		err = fmt.Errorf("ldap delete no valid ldap handler")
		return
	}
	if len(dn) == 0 {
		err = fmt.Errorf("ldap delete dn empty")
		return
	}
	req := ldap.NewDelRequest(dn, nil)
	err = l.Del(req)
	return
}

// AddEntry creates a new Entry
func AddEntry(l *ldap.Conn, dn string, attr []ldap.Attribute) (err error) {
	if l == nil {
		err = fmt.Errorf("ldap delete no valid ldap handler")
		return
	}
	if len(dn) == 0 {
		err = fmt.Errorf("ldap delete dn empty")
		return
	}
	if len(attr) == 0 {
		err = fmt.Errorf("ldap add attributes empty")
		return
	}
	req := ldap.NewAddRequest(dn, nil)
	for _, a := range attr {
		req.Attribute(a.Type, a.Vals)
	}
	err = l.Add(req)
	return
}

// ModifyAttribute add, replaces or deletes one Attribute of an Entry
func ModifyAttribute(l *ldap.Conn, dn string, modtype string, name string, values []string) (err error) {
	if l == nil {
		err = fmt.Errorf("ldap modify no valid ldap handler")
		return
	}
	if len(dn) == 0 {
		err = fmt.Errorf("ldap modify dn empty")
		return
	}
	if len(name) == 0 {
		err = fmt.Errorf("ldap modify attribute name empty")
		return
	}
	if len(values) == 0 {
		err = fmt.Errorf("ldap modify values empty")
		return
	}
	req := ldap.NewModifyRequest(dn, nil)
	switch modtype {
	case "add":
		req.Add(name, values)
	case "modify":
		req.Replace(name, values)
	case "replace":
		req.Replace(name, values)
	case "delete":
		req.Delete(name, values)
	case "increment":
		req.Increment(name, values[0])
	default:
		err = fmt.Errorf("ldap modify unknow type %s", modtype)
		return
	}
	err = l.Modify(req)
	return
}

// SetPassword changes an existing password to the given or generated value
func SetPassword(l *ldap.Conn, dn string, oldPass string, newPass string) (generatedPass string, err error) {
	passwdModReq := ldap.NewPasswordModifyRequest(dn, oldPass, newPass)
	passwdModResp, err := l.PasswordModify(passwdModReq)
	if err != nil {
		return
	}
	if newPass == "" {
		generatedPass = passwdModResp.GeneratedPassword
	}
	return
}
