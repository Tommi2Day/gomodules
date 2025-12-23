package ldaplib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldif"
)

// ApplyLDIFDir applies all LDIF files in a directory matching the pattern.
func (lc *LdapConfigType) ApplyLDIFDir(dir string, pattern string, ignore bool) error {
	if pattern == "" {
		pattern = "**/*.ldif"
	}
	files, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("failed to glob ldif files: %w", err)
	}
	for _, file := range files {
		var data []byte
		data, err = os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read ldif file %s: %w", file, err)
		}

		if err = lc.ApplyLDIF(string(data), ignore); err != nil {
			return fmt.Errorf("failed to apply ldif file %s: %w", file, err)
		}
	}
	return nil
}

func (lc *LdapConfigType) ApplyLDIF(ldifData string, ignore bool) (err error) {
	if len(ldifData) == 0 {
		err = fmt.Errorf("ldif is empty")
		return
	}
	if lc.Conn == nil {
		err = fmt.Errorf("ldap connection is nil")
		return
	}
	l, err := ldif.Parse(ldifData)
	if err != nil {
		return
	}
	for _, entry := range l.Entries {
		if err = lc.applyEntry(entry, ignore); err != nil {
			return err
		}
	}
	return nil
}

func (lc *LdapConfigType) applyEntry(entry *ldif.Entry, ignore bool) error {
	var err error
	switch {
	case entry.Entry != nil:
		add := ldap.NewAddRequest(entry.Entry.DN, nil)
		for _, attr := range entry.Entry.Attributes {
			add.Attribute(attr.Name, attr.Values)
		}
		err = lc.Conn.Add(add)
	case entry.Add != nil:
		err = lc.Conn.Add(entry.Add)
	case entry.Del != nil:
		err = lc.Conn.Del(entry.Del)
	case entry.Modify != nil:
		err = lc.Conn.Modify(entry.Modify)
	}

	if err != nil && !ignore {
		return fmt.Errorf("failed to process entry: %w", err)
	}
	return nil
}

func ExportLDIF(entries []*ldap.Entry) (ldifData string, err error) {
	ldifData = ""
	foldWith := 0
	buf := bytes.NewBuffer(nil)
	if len(entries) == 0 {
		err = fmt.Errorf("no entries to dump")
		return
	}

	err = ldif.Dump(buf, foldWith, entries)
	if err != nil {
		return
	}
	ldifData = buf.String()
	return
}
