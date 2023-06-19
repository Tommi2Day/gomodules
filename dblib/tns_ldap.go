package dblib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tommi2day/gomodules/ldaplib"

	"gopkg.in/ini.v1"

	ldap "github.com/go-ldap/ldap/v3"
	log "github.com/sirupsen/logrus"
)

// LdapServer  covers properties from one server in ldap.ora
type LdapServer struct {
	Hostname string
	Port     int
	SSLPort  int
}

// ReadLdapTns read tns entries from ldap
func ReadLdapTns(lc *ldaplib.LdapConfigType, contextDN string) (TNSEntries, error) {
	tnsEntries := make(TNSEntries)
	log.Infof("ReadLdap with OracleContext %s", contextDN)
	if lc == nil {
		err := fmt.Errorf("no valid ldap config given")
		return tnsEntries, err
	}
	ldapFilter := fmt.Sprintf("(objectClass=%s)", ldap.EscapeFilter("orclNetService"))
	result, err := lc.Search(contextDN, ldapFilter, []string{"cn", "orclNetDescString", "aliasedObjectName"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
	if err != nil {
		err = fmt.Errorf("service search returned error:%v", err)
		log.Errorf("Ldap: %v", err)
		return tnsEntries, err
	}
	count := len(result)
	log.Debugf("Found %d TNS Ldap Entries", count)
	if count == 0 {
		return tnsEntries, nil
	}
	for _, e := range result {
		cn := e.GetEqualFoldAttributeValue("cn")
		desc := e.GetEqualFoldAttributeValue("orclNetDescString")
		alias := e.GetEqualFoldAttributeValue("aliasedObjectName")
		dn := e.DN
		log.Debugf("LDAP DN=%s, CN=%s", dn, cn)

		if len(cn) > 0 && len(desc) > 0 {
			tnsEntries[strings.ToUpper(cn)] = BuildTnsEntry(dn, desc, cn)
			if len(alias) > 0 {
				tnsEntries[strings.ToUpper(alias)] = BuildTnsEntry(dn, desc, alias)
				log.Debugf("use alias %s instead of cn %s", alias, cn)
			}
		}
	}
	log.Infof("Return %d Ldap Entries", len(tnsEntries))
	return tnsEntries, nil
}

// ReadLdapOra Reads ldap ora and returns servers and context
func ReadLdapOra(path string) (ctx string, servers []LdapServer) {
	filename := path + "/ldap.ora"
	ctx = ""
	host := ""
	port := 0
	ssl := 0
	log.Debugf("Try to read ldap.ora at %s", filename)
	cfg, err := ini.InsensitiveLoad(filename)
	if err != nil {
		log.Debugf("Cannot Read ldap.ora at %s", filename)
		return
	}
	ctx = cfg.Section("").Key(strings.ToLower("DEFAULT_ADMIN_CONTEXT")).String()
	srv := cfg.Section("").Key(strings.ToLower("DIRECTORY_SERVERS")).String()
	replacer := strings.NewReplacer("(", "", ")", "", " ", "")
	srv = replacer.Replace(srv)
	srvs := strings.Split(srv, ",")
	for _, e := range srvs {
		if !strings.Contains(e, ":") {
			continue
		}
		f := strings.Split(e, ":")
		if len(f) < 2 {
			continue
		}
		host = f[0]
		port, _ = strconv.Atoi(f[1])
		if strings.TrimSpace(host) == "" {
			continue
		}
		ssl = 0
		if len(f) > 2 {
			ssl, _ = strconv.Atoi(f[2])
		}
		server := LdapServer{Hostname: host, Port: port, SSLPort: ssl}
		servers = append(servers, server)
	}
	log.Debugf("ReadLdapOra CTX: %s, Servers %v", ctx, servers)
	return
}

// GetOracleContext retrieve next OracleContext Object from LDAP
func GetOracleContext(lc *ldaplib.LdapConfigType, basedn string) (contextDN string, err error) {
	ldapFilter := fmt.Sprintf("(objectClass=%s)", ldap.EscapeFilter("orclContext"))
	result, err := lc.Search(basedn, ldapFilter, []string{"DN"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
	if err != nil {
		err = fmt.Errorf("context search returned error:%v", err)
		log.Errorf("Ldap: %v", err)
		return "", err
	}
	if len(result) == 0 {
		err = fmt.Errorf("oracle context not found")
		log.Errorf("Ldap: %v", err)
		return "", err
	}
	contextDN = result[0].DN
	log.Debugf("ContextDN=%s", contextDN)
	return
}

// AddLdapTNSEntry add new entry in LDAP
func AddLdapTNSEntry(lc *ldaplib.LdapConfigType, context string, alias string, desc string) (err error) {
	log.Debugf("Add Ldap Entry for alias %s", alias)
	var attributes []ldap.Attribute
	name := strings.ToUpper(alias)
	dn := fmt.Sprintf("cn=%s,%s", name, context)
	attributes = append(attributes, ldap.Attribute{Type: "objectClass", Vals: []string{"top", "orclNetService"}})
	attributes = append(attributes, ldap.Attribute{Type: "cn", Vals: []string{name}})
	attributes = append(attributes, ldap.Attribute{Type: "orclNetDescString", Vals: []string{desc}})
	err = lc.AddEntry(dn, attributes)
	return
}

// ModifyLdapTNSEntry replace n existing entry
func ModifyLdapTNSEntry(lc *ldaplib.LdapConfigType, dn string, alias string, desc string) (err error) {
	log.Debugf("Modify Ldap Entry for alias %s", alias)
	err = lc.ModifyAttribute(dn, "replace", "orclNetDescString", []string{desc})
	return
}

// DeleteLdapTNSEntry removes an Entry
func DeleteLdapTNSEntry(lc *ldaplib.LdapConfigType, dn string, alias string) (err error) {
	log.Debugf("Delete Ldap Entry for alias %s", alias)
	err = lc.DeleteEntry(dn)
	return
}
