// Package dblib collection of db func
package dblib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tommi2day/gomodules/common"

	"gopkg.in/ini.v1"

	log "github.com/sirupsen/logrus"
)

// TNSAddress holds  host/port of an address section
type TNSAddress struct {
	Host string
	Port string
}

// TNSEntry structure for holding one entry from tnsnames.ora
type TNSEntry struct {
	Name    string
	Desc    string
	File    string
	Service string
	Servers []TNSAddress
}

// TNSEntries Map of tns entries
type TNSEntries map[string]TNSEntry

// CheckTNSadmin verify TNS-Admin Settings
func CheckTNSadmin(tnsadmin string) (dn string, err error) {
	dn, err = filepath.Abs(tnsadmin)
	if err != nil {
		log.Errorf("Cannot get absolute name of '%s'", tnsadmin)
		return
	}
	_, err = os.Stat(dn)
	if os.IsNotExist(err) {
		log.Errorf("TNSAdmin directory '%s' not found", dn)
		return
	}
	log.Debugf("TNS_ADMIN absolute path: %s", dn)
	sq := filepath.Join(dn, "sqlnet.ora")
	_, err = os.Stat(sq)
	if os.IsNotExist(err) {
		log.Warnf("no sqlnet.ora in TNSAdmin directory '%s' found", dn)
	}
	return
}

// BuildTnsEntry build map for entry
func BuildTnsEntry(filename string, desc string, tnsAlias string) TNSEntry {
	var service = ""
	reService := regexp.MustCompile(`(?mi)SERVICE_NAME\s*=\s*([\w.]+)`)
	s := reService.FindStringSubmatch(desc)
	if len(s) > 1 {
		service = s[1]
	}
	servers := getServers(desc)
	entry := TNSEntry{Name: tnsAlias, Desc: desc, File: filename, Service: service, Servers: servers}
	log.Debugf("found TNS Alias %s", tnsAlias)
	return entry
}

// ReadSqlnetOra reads a sqlnet.ora and returns default domain and names path
func ReadSqlnetOra(path string) (domain string, namesPath []string) {
	filename := path + "/sqlnet.ora"
	domain = ""
	cfg, err := ini.InsensitiveLoad(filename)
	if err != nil {
		log.Debugf("cannot Read %s", filename)
		return
	}
	// all keys are lowwer case
	domain = cfg.Section("").Key(strings.ToLower("NAMES.DEFAULT_DOMAIN")).String()
	names := cfg.Section("").Key(strings.ToLower("NAMES.DIRECTORY_PATH")).String()
	replacer := strings.NewReplacer("(", "", ")", "", " ", "")
	names = replacer.Replace(names)
	namesPath = strings.Split(names, ",")
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
	domain, _ := ReadSqlnetOra(tnsDir)

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
	for _, line := range content {
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
				tnsEntries[tnsAlias] = BuildTnsEntry(filename, desc, tnsAlias)
			}
			// new entry
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
		tnsEntries[tnsAlias] = BuildTnsEntry(filename, desc, tnsAlias)
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

func tnsSanity(entries TNSEntries) (tnsEntries TNSEntries, deletes int) {
	// sanity check
	d := 0
	for k, e := range entries {
		se := 0
		if len(e.Name) == 0 {
			log.Errorf("Entry %s has no name set", k)
			se++
		}
		if len(e.Desc) == 0 {
			log.Errorf("Entry %s has no description set", k)
			se++
		}
		if len(e.Service) == 0 {
			log.Errorf("Entry %s has no Service set", k)
			se++
		}
		if len(e.Servers) == 0 {
			log.Errorf("Entry %s has no Oracle Host set", k)
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
	re := regexp.MustCompile(`(?m)HOST\s*=\s*([\w.]+)\s*\)\s*\(\s*PORT\s*=\s*(\d+)`)
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
