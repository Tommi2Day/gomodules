package dblib

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tommi2day/gomodules/ldaplib"

	"gopkg.in/ini.v1"

	ldap "github.com/go-ldap/ldap/v3"
	log "github.com/sirupsen/logrus"
)

const (
	sOK   = "ok"
	sNew  = "new"
	sMod  = "mod"
	sDel  = "del"
	sSkip = "skip"
)

// TWorkStatus structure to handover statistics
type TWorkStatus map[string]int

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

// buildstatus creates ops task map to handle
func buildStatusMap(lc *ldaplib.LdapConfigType, tnsEntries TNSEntries, contextDN string) (TNSEntries, map[string]string, error) {
	var alias string
	ldapstatus := map[string]string{}

	ldapTNS, err := ReadLdapTns(lc, contextDN)
	if err != nil {
		return nil, ldapstatus, err
	}
	for _, a := range ldapTNS {
		alias = a.Name
		ldapstatus[alias] = ""
		log.Debugf("prepare status for LDAP Alias  %s", alias)
	}

	for _, t := range tnsEntries {
		alias = t.Name
		l, valid := ldapTNS[alias]
		if valid {
			comp := strings.Compare(l.Desc, t.Desc)
			if comp == 0 {
				ldapstatus[alias] = sOK
				log.Debugf("TNS Alias %s exists in LDAP and is equal ->OK", alias)
				continue
			}
			ldapstatus[alias] = sMod
			log.Debugf("TNS Alias %s exists in LDAP, but description changed ->MOD", alias)
		} else {
			ldapstatus[alias] = sNew
			log.Debugf("TNS Alias %s missed in LDAP ->NEW", alias)
		}
	}
	return ldapTNS, ldapstatus, err
}

// WriteLdapTns writes a set of TNS entries to Ldap
func WriteLdapTns(lc *ldaplib.LdapConfigType, tnsEntries TNSEntries, domain string, contextDN string) (TWorkStatus, error) {
	var ldapstatus map[string]string
	var ldapTNS TNSEntries
	var alias string
	var err error
	workStatus := make(TWorkStatus)
	workStatus[sOK] = 0
	workStatus[sMod] = 0
	workStatus[sNew] = 0
	workStatus[sDel] = 0
	workStatus[sSkip] = 0

	log.Infof("Update LDAP Context %s with %d tnsnames.ora entries using domain %s", contextDN, len(tnsEntries), domain)
	ldapTNS, ldapstatus, err = buildStatusMap(lc, tnsEntries, contextDN)
	status := ""

	// sort candidates
	sortedAlias := make([]string, 0, len(ldapstatus))
	for k := range ldapstatus {
		sortedAlias = append(sortedAlias, k)
	}
	sort.Strings(sortedAlias)
	for _, alias = range sortedAlias {
		status = ldapstatus[alias]
		switch status {
		case sOK:
			log.Debugf("Alias %s unchanged", alias)
			workStatus[sOK]++
		case sNew:
			tnsEntry, valid := tnsEntries[alias]
			if !valid {
				log.Warnf("Skip add invalid tns alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			err = AddLdapTNSEntry(lc, contextDN, tnsEntry.Name, tnsEntry.Desc)
			if err != nil {
				log.Warnf("Add %s failed: %v", tnsEntry.Name, err)
				workStatus[sSkip]++
				continue
			}
			workStatus[sNew]++
			log.Debugf("Alias %s added", tnsEntry.Name)
		case sMod:
			// delete and add
			ldapEntry, valid := ldapTNS[alias]
			if !valid {
				log.Warnf("Skip modify invalid ldap alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			dn := ldapEntry.File
			tnsEntry, valid := tnsEntries[alias]
			if !valid {
				log.Warnf("Skip modify invalid tns alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			err = ModifyLdapTNSEntry(lc, dn, tnsEntry.Name, tnsEntry.Desc)
			if err != nil {
				log.Warnf("Modify %s failed: %v", tnsEntry.Name, err)
				workStatus[sSkip]++
			} else {
				log.Debugf("Alias %s replaced", tnsEntry.Name)
				workStatus[sMod]++
			}

		case "":
			ldapEntry, valid := ldapTNS[alias]
			if !valid {
				log.Warnf("Skip delete invalid ldap alias %s", alias)
				workStatus[sSkip]++
				continue
			}
			dn := ldapEntry.File
			err = DeleteLdapTNSEntry(lc, dn, alias)
			if err != nil {
				log.Warnf("Delete %s failed: %v", alias, err)
				workStatus[sSkip]++
			} else {
				log.Debugf("Alias %s deleted", alias)
				workStatus[sDel]++
			}
		}
	}
	log.Infof("%d TNS entries unchanged,%d new written, %d modified, %d deleted and %d skipped because of errors",
		workStatus[sOK], workStatus[sNew], workStatus[sMod], workStatus[sDel], workStatus[sSkip])
	return workStatus, err
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
