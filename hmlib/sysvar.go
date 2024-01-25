package hmlib

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// SysVarEndpoint is the endpoint for a given system variable
const SysVarEndpoint = "/addons/xmlapi/sysvar.cgi"

// SysVarListEndpoint is the endpoint for the system variables list
const SysVarListEndpoint = "/addons/xmlapi/sysvarlist.cgi"

// SysvarListResponse is a list of system variables returned by API
type SysvarListResponse struct {
	XMLName     xml.Name      `xml:"systemVariables"`
	SysvarEntry []SysVarEntry `xml:"systemVariable"`
}

// SysVarEntry is a single system variable in SysvarListResponse
type SysVarEntry struct {
	XMLName    xml.Name `xml:"systemVariable"`
	Name       string   `xml:"Name,attr"`
	Variable   string   `xml:"variable,attr"`
	Value      string   `xml:"value,attr"`
	ValueList  string   `xml:"value_list,attr"`
	ValueText  string   `xml:"value_text,attr"`
	IseID      string   `xml:"ise_id,attr"`
	Min        string   `xml:"min,attr"`
	Max        string   `xml:"max,attr"`
	Unit       string   `xml:"unit,attr"`
	Type       string   `xml:"type,attr"`
	Subtype    string   `xml:"subtype,attr"`
	Timestamp  string   `xml:"timestamp,attr"`
	ValueName0 string   `xml:"value_name_0,attr"`
	ValueName1 string   `xml:"value_name_1,attr"`
}

const pmTrue = "true"

// SysVarIDMap is a map of system variables by ID
var SysVarIDMap = map[string]SysVarEntry{}

// String returns a string representation of the system variable list
func (e SysVarEntry) String() string {
	return fmt.Sprintf("ID:%s, %s= %s (%s),  ts %s\n", e.IseID, e.Name, e.Value, e.Unit, common.FormatUnixtsString(e.Timestamp, "2006-01-02 15:04:05"))
}

// GetSysvar returns a single system variable
func GetSysvar(sysvarIDs []string, text bool) (result SysvarListResponse, err error) {
	log.Debug("sysvars called")
	var parameter = make(map[string]string)
	if len(sysvarIDs) > 0 {
		parameter["ise_id"] = strings.Join(sysvarIDs, ",")
	}
	if text {
		parameter["text"] = pmTrue
	}
	err = QueryAPI(SysVarEndpoint, &result, parameter)
	log.Debugf("getSysvars returned %d entries", len(result.SysvarEntry))
	return
}

// GetSysvarList returns the list of system variables
func GetSysvarList(text bool) (err error) {
	var result SysvarListResponse
	var parameter = make(map[string]string)
	log.Debug("sysvarlist called")
	if text {
		parameter["text"] = pmTrue
	}
	err = QueryAPI(SysVarListEndpoint, &result, parameter)

	for _, e := range result.SysvarEntry {
		SysVarIDMap[e.IseID] = e
		AllIds[e.IseID] = IDMapEntry{e.IseID, e.Name, "Sysvar", e}
	}

	log.Debugf("getSysvarList returned %d entries", len(result.SysvarEntry))
	return
}
