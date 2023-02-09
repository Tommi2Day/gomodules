// Package dblib collection of db func
package dblib

import (
	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type address struct {
	Host string
	Port string
}

// TNSEntry structure for holding one entry from tnsnames.ora
type TNSEntry struct {
	Name    string
	Desc    string
	File    string
	Service string
	Servers []address
}

// TNSEntries Map of tns entries
type TNSEntries map[string]TNSEntry

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

// GetTnsnames map tnsnames.ora entries to an readable structure
func GetTnsnames(filename string, recursiv bool) (TNSEntries, error) {
	var tnsEntries = make(TNSEntries)
	var err error
	var content []string

	var reIfile = regexp.MustCompile(`(?im)^IFILE\s*=\s*(.*)$`)
	var reNewEntry = regexp.MustCompile(`(?im)^([\w.]+)\s*=(.*)`)
	var tnsAlias = ""

	var desc = ""

	// change to current file
	wd, _ := os.Getwd()
	log.Debugf("DEBUG: GetTns use %s, wd=%s", filename, wd)
	err = common.ChdirToFile(filename)
	if err != nil {
		log.Errorf("Cannot chdir to %s", filename)
		return tnsEntries, err
	}

	// use basename from filename to read as i am in this directory
	f := filepath.Base(filename)
	content, _ = common.ReadFileByLine(f)

	// loop through lines
	for _, line := range content {
		if checkSkip(line) {
			continue
		}

		// find and load ifiles
		ifile := reIfile.FindStringSubmatch(line)
		if len(ifile) > 0 {
			fn := ifile[1]
			ifileEntries, err := getIfile(fn, recursiv)
			if err == nil {
				for k, v := range ifileEntries {
					tnsEntries[k] = v
				}
			}
			continue
		}

		// find new entry
		newEntry := reNewEntry.FindStringSubmatch(line)
		i := len(newEntry)
		if i > 0 {
			// save previous entry
			if len(tnsAlias) > 0 && len(desc) > 0 {
				tnsEntries[tnsAlias] = buildEntry(filename, desc, tnsAlias)
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
		tnsEntries[tnsAlias] = buildEntry(filename, desc, tnsAlias)
	}

	// chdir back
	_ = os.Chdir(wd)
	return tnsEntries, err
}

// read ifile recursive
func getIfile(filename string, recursiv bool) (entries TNSEntries, err error) {
	wd, _ := os.Getwd()
	log.Debugf("read ifile %s, wd=%s", filename, wd)
	entries, err = GetTnsnames(filename, recursiv)
	return
}

// build map for entry
func buildEntry(filename string, desc string, tnsAlias string) TNSEntry {
	var service = ""
	reService := regexp.MustCompile(`(?mi)SERVICE_NAME\s*=\s*([\w.]+)`)
	s := reService.FindStringSubmatch(desc)
	if len(s) > 1 {
		service = s[1]
	}
	entry := TNSEntry{Name: tnsAlias, Desc: desc, File: filename, Service: service, Servers: getServers(desc)}
	log.Debugf("found TNS Alias %s", tnsAlias)
	return entry
}

// extract address part
func getServers(tnsDesc string) (servers []address) {
	re := regexp.MustCompile(`(?m)HOST\s*=\s*([\w.]+)\s*\)\s*\(\s*PORT\s*=\s*(\d+)`)
	match := re.FindAllStringSubmatch(tnsDesc, -1)
	for _, a := range match {
		if len(a) > 1 {
			host := a[1]
			port := a[2]
			servers = append(servers, address{
				Host: host, Port: port,
			})
			log.Debugf("parsed Host: %s, Port %s", host, port)
		}
	}
	return
}

// GetDefaultDomain extract names_default_domain from sqlnet.ora
func GetDefaultDomain(path string) (domain string) {
	filename := "sqlnet.ora"
	if path != "" {
		filename = path + "/" + filename
	}
	content, err := common.ReadFileToString(filename)
	if err != nil {
		log.Debugf("Cannot read %s, assume no default domain", filename)
		return ""
	}
	reg := regexp.MustCompile((`(?im)^NAMES.DEFAULT_DOMAIN\s*=\s*([\w.]*)`))
	result := reg.FindStringSubmatch(content)
	if len(result) == 0 {
		log.Debugf("no default domain defined in %s", filename)
		return ""
	}
	domain = result[1]
	log.Infof("default domain: %s", domain)
	return domain
}
