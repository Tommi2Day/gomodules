// Package hmlib is a library for interfacing with the HomeMatic XML API
package hmlib

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
