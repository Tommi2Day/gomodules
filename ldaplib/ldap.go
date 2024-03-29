// Package ldaplib collects ldap related functions
package ldaplib

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	ldap "github.com/go-ldap/ldap/v3"
)

// LdapConfigType helds config properties
type LdapConfigType struct {
	Server   string
	Port     int
	URL      string
	TLS      bool
	Insecure bool
	BaseDN   string
	Timeout  int // in second
	Conn     *ldap.Conn
}

// NewConfig defines common connection parameter
func NewConfig(server string, port int, tls bool, insecure bool, basedn string, timeout int) *LdapConfigType {
	ldapConfig := LdapConfigType{}
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
	ldapConfig.Timeout = timeout
	return &ldapConfig
}

// Connect will authorize to the ldap server
func (lc *LdapConfigType) Connect(bindDN string, bindPassword string) (err error) {
	l := lc.Conn
	if l != nil {
		_ = l.Close()
		l = nil
	}

	// set timeout
	ldap.DefaultTimeout = time.Duration(lc.Timeout) * time.Second

	// You can also use IP instead of FQDN
	if lc.Insecure {
		//nolint gosec
		l, err = ldap.DialURL(lc.URL, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	} else {
		l, err = ldap.DialURL(lc.URL)
	}

	if err != nil {
		return
	}
	if len(bindDN) == 0 {
		err = l.UnauthenticatedBind("")
	} else {
		err = l.Bind(bindDN, bindPassword)
	}
	if err != nil {
		return
	}
	lc.Conn = l
	return
}

// Search do a search on ldap
func (lc *LdapConfigType) Search(baseDN string, filter string, attributes []string, scope int, deref int) (entries []*ldap.Entry, err error) {
	var result *ldap.SearchResult
	l := lc.Conn
	if l == nil {
		err = fmt.Errorf("ldap search: not connected")
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
func (lc *LdapConfigType) DeleteEntry(dn string) (err error) {
	l := lc.Conn
	if l == nil {
		err = fmt.Errorf("ldap delete: not connected")
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
func (lc *LdapConfigType) AddEntry(dn string, attr []ldap.Attribute) (err error) {
	l := lc.Conn
	if l == nil {
		err = fmt.Errorf("ldap add: not connected")
		return
	}
	if len(dn) == 0 {
		err = fmt.Errorf("ldap add: dn empty")
		return
	}
	if len(attr) == 0 {
		err = fmt.Errorf("ldap add: attributes empty")
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
func (lc *LdapConfigType) ModifyAttribute(dn string, modtype string, name string, values []string) (err error) {
	l := lc.Conn
	if l == nil {
		err = fmt.Errorf("ldap modify: not connected")
		return
	}
	if len(dn) == 0 {
		err = fmt.Errorf("ldap modify: dn empty")
		return
	}
	if len(name) == 0 {
		err = fmt.Errorf("ldap modify: attribute name empty")
		return
	}
	if len(values) == 0 {
		err = fmt.Errorf("ldap modify: values empty")
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
func (lc *LdapConfigType) SetPassword(dn string, oldPass string, newPass string) (generatedPass string, err error) {
	// all parameter can be empty
	l := lc.Conn
	if l == nil {
		err = fmt.Errorf("ldap delete: not connected")
		return
	}
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

// RetrieveEntry returns the first entry found for the given DN
func (lc *LdapConfigType) RetrieveEntry(dn string, filter string, fields string) (entry *ldap.Entry, err error) {
	if len(dn) == 0 {
		err = fmt.Errorf("ldap lookup: dn empty")
		return
	}
	if len(filter) == 0 {
		filter = "(objectclass=*)"
	}
	f := strings.Split(fields, ",")
	if f[0] == "" {
		f = []string{"*"}
	}
	entries, err := lc.Search(dn, filter, f, ldap.ScopeBaseObject, ldap.DerefInSearching)
	if err != nil {
		err = fmt.Errorf("ldap search for %s returned error %v", dn, err)
		return
	}
	if len(entries) == 0 {
		err = fmt.Errorf("ldap search for %s returned no entry", dn)
		return
	}
	entry = entries[0]
	return
}

// HasAttribute checks if the given entry has the given attribute
func HasAttribute(entry *ldap.Entry, attribute string) bool {
	if entry == nil {
		return false
	}
	if len(entry.Attributes) == 0 {
		return false
	}
	attribute = strings.ToLower(attribute)
	for _, a := range entry.Attributes {
		name := strings.ToLower(a.Name)
		if name == attribute {
			return true
		}
	}
	return false
}

// HasObjectClass checks if the given entry has the given objectClass
func HasObjectClass(entry *ldap.Entry, objectClass string) bool {
	if entry == nil {
		return false
	}
	if len(entry.Attributes) == 0 {
		return false
	}
	objectClass = strings.ToLower(objectClass)
	oc := entry.GetAttributeValues("objectClass")
	for _, c := range oc {
		c = strings.ToLower(c)
		if c == objectClass {
			return true
		}
	}
	return false
}
