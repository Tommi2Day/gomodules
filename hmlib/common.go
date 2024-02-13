// Package hmlib is a library for interfacing with the HomeMatic XML API
package hmlib

// IDMapEntry contains the objects for any ID
type IDMapEntry struct {
	IseID     string
	Name      string
	EntryType string
	Entry     any
}

// AllIDs is a map of all iseIDs to IDMapEntry
var AllIDs = map[string]IDMapEntry{}

// NameIDMap is a map of all names to iseIDs
var NameIDMap = map[string]string{}
