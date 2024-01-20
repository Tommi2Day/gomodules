// Package hmlib is a library for interfacing with the HomeMatic XML API
package hmlib

import (
	"strconv"
	"time"
)

// IDMapEntry contains the objects for any ID
type IDMapEntry struct {
	IseID     string
	Name      string
	EntryType string
	Entry     any
}

// AllIds is a map of all iseIDs to IDMapEntry
var AllIds = map[string]IDMapEntry{}

// NameIDMap is a map of all names to iseIDs
var NameIDMap = map[string]string{}

// FormatUnixtsString converts a unix timestamp string to a human readable
func FormatUnixtsString(ts string) string {
	timestamp, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ts
	}
	return time.Unix(timestamp, 0).Format(time.RFC3339)
}
