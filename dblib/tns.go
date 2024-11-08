// Package dblib collection of db func
package dblib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
)

// TNSAddress holds  host/port of an address section
type TNSAddress struct {
	Host string
	Port string
}

// TNSEntry structure for holding one entry from tnsnames.ora
type TNSEntry struct {
	Name     string
	Desc     string
	Location string
	Service  string
	Servers  []TNSAddress
}

// TNSEntries Map of tns entries
type TNSEntries map[string]TNSEntry
type TNSSSL struct {
	WalletLocation      string
	ClientAthentication bool
	ServerDNMatch       bool
	Ciphers             string
	Version             string
}

const ldapora = "ldap.ora"
const sqlnetora = "sqlnet.ora"

// ModifyJDBCTransportConnectTimeout if true, TRANSPORT_CONNECT_TIMEOUT is modified to be *1000
var ModifyJDBCTransportConnectTimeout = true
var TNSSSLconfig TNSSSL

// CheckTNSadmin verify TNS-Admin Settings
func CheckTNSadmin(tnsadmin string) (tnsnames string, err error) {
	tnsnames, err = filepath.Abs(tnsadmin)
	if err != nil {
		log.Errorf("Cannot get absolute name of '%s'", tnsadmin)
		return
	}
	_, err = os.Stat(tnsnames)
	if os.IsNotExist(err) {
		log.Errorf("TNSAdmin directory '%s' not found", tnsnames)
		return
	}
	log.Debugf("TNS_ADMIN absolute path: %s", tnsnames)
	sqlnet := filepath.Join(tnsnames, "sqlnet.ora")
	_, err = os.Stat(sqlnet)
	if os.IsNotExist(err) {
		log.Warnf("no sqlnet.ora in TNSAdmin directory '%s' found", tnsnames)
	}
	return
}

// BuildTnsEntry build map for entry
func BuildTnsEntry(location string, desc string, tnsAlias string) TNSEntry {
	var service = ""
	reService := regexp.MustCompile(`(?mi)(?:SERVICE_NAME|SID)\s*=\s*([\w.]+)`)
	s := reService.FindStringSubmatch(desc)
	if len(s) > 1 {
		service = s[1]
	}
	servers := getServers(desc)
	entry := TNSEntry{Name: tnsAlias, Desc: desc, Location: location, Service: service, Servers: servers}
	log.Debugf("Build Entry for %s", tnsAlias)
	return entry
}

// ReadSQLNetOra reads a sqlnet.ora and returns default domain and names path
func ReadSQLNetOra(filePath string) (domain string, namesPath []string, sslInfo TNSSSL) {
	filename := path.Join(filePath, sqlnetora)
	domain = ""
	sslCfg := TNSSSL{WalletLocation: "", ClientAthentication: false, ServerDNMatch: false, Ciphers: "", Version: ""}

	content, err := common.ReadFileToString(filename)
	if err != nil {
		log.Debugf("Cannot read sqlnet.ora as %s: %v", filename, err)
		return
	}
	// ^"(.+?)(?<!\\)"\s*=\s*"([\s\S]*?)(?<!\\)";
	// all keys are lowwer case
	// ini cannt parse multiline entries, so we use string matching

	// Regular expression to match key-value pairs
	re := regexp.MustCompile(`(?m)^\s*(\w+(?:\.\w+)*)\s*=\s*(.+?)\s*$`)
	matches := re.FindAllStringSubmatch(content, -1)
	names := ""
	for _, match := range matches {
		key := strings.ToLower(match[1])
		value := strings.Trim(match[2], `"()`)
		value = strings.ReplaceAll(value, " ", "")
		switch key {
		case "names.default_domain":
			domain = value
		case "names.directory_path":
			namesPath = strings.Split(value, ",")
		case "ssl_cipher_suites":
			sslCfg.Ciphers = value
		case "ssl_version":
			sslCfg.Version = value
		case "ssl_server_dn_match":
			sslCfg.ServerDNMatch, _ = regexp.MatchString(`(?i)YES|ON|TRUE|1`, value)
		case "ssl_client_authentication":
			sslCfg.ClientAthentication, _ = strconv.ParseBool(value)
		}
	}

	// Handle WALLET_LOCATION separately as it's multi-line
	reWallet := regexp.MustCompile(`WALLET_LOCATION\s*=\s*\(\s*SOURCE\s*=\s*\(\s*METHOD\s*=\s*FILE\s*\)\s*\(\s*METHOD_DATA\s*=\s*\(\s*DIRECTORY\s*=\s*"([^"]*)"`)
	if walletMatch := reWallet.FindStringSubmatch(content); len(walletMatch) > 1 {
		sslCfg.WalletLocation = walletMatch[1]
	}
	// all keys are lowwer case
	log.Debugf("parsed %s, domain=%s, names filePath=%s, Wallet=%s", filename, domain, names, sslCfg.WalletLocation)
	sslInfo = sslCfg
	return
}

// GetEntry matches given string to tns entries using with domain part and without
func GetEntry(alias string, entries TNSEntries, domain string) (e TNSEntry, ok bool) {
	match, _ := regexp.MatchString(`^\w+\.`, alias)
	if len(domain) > 0 {
		if match {
			e, ok = entries[strings.ToUpper(alias)] // full qualified
		} else {
			e, ok = entries[strings.ToUpper(alias)+"."+strings.ToUpper(domain)] // short alias+domain
		}
		return
	}
	// no domain, only full match accepted
	e, ok = entries[strings.ToUpper(alias)]
	return
}

// GetTnsnames map tnsnames.ora entries to a readable structure
func GetTnsnames(filename string, recursiv bool) (TNSEntries, string, error) {
	var tnsEntries = make(TNSEntries)
	var err error
	var content []string
	var reIfile = regexp.MustCompile(`(?im)^IFILE\s*=\s*(.*)$`)
	var reNewEntry = regexp.MustCompile(`(?im)^([\w.]+)\s*=(.*)`)
	var tnsAlias = ""
	var desc = ""

	// try to find sqlnet ora and read domain
	tnsDir := filepath.Dir(filename)
	domain, _, _ := ReadSQLNetOra(tnsDir)

	// change to current tns file
	wd, _ := os.Getwd()
	log.Debugf("DEBUG: GetTns use %s, wd=%s", filename, wd)
	err = common.ChdirToFile(filename)
	if err != nil {
		log.Errorf("Cannot chdir to %s", filename)
	}

	// use basename from filename to read as I am in this directory
	f := filepath.Base(filename)
	content, _ = common.ReadFileByLine(f)

	// loop through lines
	l := 0
	location := filename
	for _, line := range content {
		l++
		if checkSkip(line) {
			continue
		}

		// find and load ifiles
		ifile := reIfile.FindStringSubmatch(line)
		var ifileEntries TNSEntries
		if len(ifile) > 0 {
			fn := ifile[1]
			ifileEntries, err = getIfile(fn, recursiv)
			for k, v := range ifileEntries {
				tnsEntries[k] = v
			}
			continue
		}

		// find new entry
		newEntry := reNewEntry.FindStringSubmatch(line)
		i := len(newEntry)
		if i > 0 {
			// save previous entry
			if len(tnsAlias) > 0 && len(desc) > 0 {
				tnsEntries[tnsAlias] = BuildTnsEntry(location, desc, tnsAlias)
			}
			// new entry
			location = fmt.Sprintf("%s Line: %d", filename, l)
			tnsAlias = strings.ToUpper(newEntry[1])
			if i > 2 {
				desc = newEntry[2] + "\n"
			}
		} else {
			desc += line
		}
	}

	// save last entry
	if len(tnsAlias) > 0 && len(desc) > 0 {
		tnsEntries[tnsAlias] = BuildTnsEntry(location, desc, tnsAlias)
	}

	// sanity check
	d := 0
	tnsEntries, d = tnsSanity(tnsEntries)
	if d > 0 {
		err = fmt.Errorf("%s had %d parsing errors", filename, d)
	}
	// chdir back
	_ = os.Chdir(wd)
	return tnsEntries, domain, err
}

// GetJDBCUrl build a jdbc url from a tns description
func GetJDBCUrl(desc string) (out string, err error) {
	var pattern *regexp.Regexp
	repl := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "")
	url := repl.Replace(desc)

	// handle transport connect timeout to be *1000
	if ModifyJDBCTransportConnectTimeout {
		pattern = regexp.MustCompile("(?i)TRANSPORT_CONNECT_TIMEOUT=([0-9]+)")
		subStr := pattern.FindStringSubmatch(url)
		if len(subStr) > 1 {
			tcval := 0
			old := subStr[0]
			tc := subStr[1]
			tcval, err = common.GetIntVal(tc)
			if err == nil {
				if tcval > 0 && tcval < 1000 {
					tcval *= 1000
					tc = fmt.Sprintf("TRANSPORT_CONNECT_TIMEOUT=%d", tcval)
					url = strings.ReplaceAll(url, old, tc)
					log.Debugf("set TRANSPORT_CONNECT_TIMEOUT to %d", tcval)
				}
			}
		}
	}

	if TNSSSLconfig.WalletLocation != "" {
		url = fmt.Sprintf("%s?WALLET_LOCATION=%s", url, TNSSSLconfig.WalletLocation)
	}
	err = nil
	out = fmt.Sprintf("jdbc:oracle:thin:@%s", url)
	return
}

// tnsSanity sanity check tns entries
func tnsSanity(entries TNSEntries) (tnsEntries TNSEntries, deletes int) {
	// sanity check
	d := 0
	for k, e := range entries {
		se := 0
		if len(e.Name) == 0 {
			log.Errorf("Sanity: Entry %s has no name set", k)
			se++
		}
		if len(e.Desc) == 0 {
			log.Errorf("Sanity: Entry %s has no description set", k)
			se++
		}
		if len(e.Service) == 0 {
			log.Errorf("Sanity: Entry %s has no SERVICE_NAME or SID set", k)
			se++
		}
		if len(e.Servers) == 0 {
			log.Errorf("Sanity: Entry %s has no SERVER set", k)
			se++
		}
		if se > 0 {
			delete(entries, k)
			d++
		}
	}
	return entries, d
}

// checkSkip returns if a line might be skipped
func checkSkip(line string) (skip bool) {
	skip = true
	found := false
	reEmpty := regexp.MustCompile(`\S`)
	reComment := regexp.MustCompile(`^#`)
	found = reEmpty.MatchString(line)
	if !found {
		return
	}
	found = reComment.MatchString(line)
	if found {
		return
	}
	skip = false
	return
}

// getIfile read ifile recursive
func getIfile(filename string, recursiv bool) (entries TNSEntries, err error) {
	wd, _ := os.Getwd()
	log.Debugf("read ifile %s, wd=%s", filename, wd)
	entries, _, err = GetTnsnames(filename, recursiv)
	return
}

// getServers extract TNSAddress part
func getServers(tnsDesc string) (servers []TNSAddress) {
	re := regexp.MustCompile(`(?m)HOST\s*=\s*([\w\-_.]+)\s*\)\s*\(\s*PORT\s*=\s*(\d+)`)
	match := re.FindAllStringSubmatch(tnsDesc, -1)
	for _, a := range match {
		if len(a) > 1 {
			host := a[1]
			port := a[2]
			servers = append(servers, TNSAddress{
				Host: host, Port: port,
			})
			log.Debugf("parsed Host: %s, Port %s", host, port)
		}
	}
	return
}
