package hmlib

import (
	"encoding/xml"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MasterValueEndpoint is the endpoint for retrieving master value for a device
const MasterValueEndpoint = "/addons/xmlapi/mastervalue.cgi"

// MasterValueChangeEndpoint is the endpoint for changing master value for a device
const MasterValueChangeEndpoint = "/addons/xmlapi/mastervaluechange.cgi"

// MasterValues is a list of devices with their master values
type MasterValues struct {
	XMLName            xml.Name            `xml:"mastervalue"`
	MasterValueDevices []MasterValueDevice `xml:"device"`
}

// MasterValueDevice is a single device with its master values
type MasterValueDevice struct {
	XMLName     xml.Name           `xml:"device"`
	Name        string             `xml:"name,attr"`
	IseID       string             `xml:"ise_id,attr"`
	DeviceType  string             `xml:"device_type,attr"`
	MasterValue []MasterValueEntry `xml:"mastervalue"`
	Error       string             `xml:"error"`
	Content     string             `xml:",chardata"`
}

// MasterValueEntry is a single entry of a master value
type MasterValueEntry struct {
	XMLName xml.Name `xml:"mastervalue"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

// GetMasterValues returns the master values of the given devices
func GetMasterValues(deviceIDs []string, requestedNames []string) (result MasterValues, err error) {
	var parameter = map[string]string{}
	log.Debug("getmastervalue called")
	if len(deviceIDs) == 0 {
		err = fmt.Errorf("no device id given")
		return
	}

	parameter["device_id"] = strings.Join(deviceIDs, ",")
	if len(requestedNames) > 0 {
		parameter["requested_names"] = strings.Join(requestedNames, ",")
	}

	err = QueryAPI(MasterValueEndpoint, &result, parameter)
	if err != nil {
		err = fmt.Errorf("value query Error id %v", err)
		return
	}
	for _, e := range result.MasterValueDevices {
		if len(e.Content) > 0 {
			c := strings.TrimSpace(e.Content)
			if len(c) > 0 {
				err = fmt.Errorf("device error for id %s: %s", e.IseID, c)
				return
			}
		}
	}
	log.Debugf("getmastervalues returned: %v", result)
	return
}

// ChangeMasterValues changes the master values of the given devices
func ChangeMasterValues(deviceIDs []string, names []string, values []string) (result MasterValues, err error) {
	var parameter = map[string]string{}
	log.Debug("getmastervalue called")
	if len(deviceIDs) == 0 {
		err = fmt.Errorf("no device id given")
		return
	}

	parameter["device_id"] = strings.Join(deviceIDs, ",")
	if len(names) > 0 {
		parameter["name"] = strings.Join(names, ",")
	}
	if len(values) > 0 {
		parameter["value"] = strings.Join(values, ",")
	}

	err = QueryAPI(MasterValueChangeEndpoint, &result, parameter)
	if err != nil {
		err = fmt.Errorf("value query Error id %v", err)
		return
	}
	for _, e := range result.MasterValueDevices {
		if len(e.Content) > 0 {
			c := strings.TrimSpace(e.Content)
			if len(c) > 0 {
				err = fmt.Errorf("device error for id %s: %s", e.IseID, c)
				return
			}
		}
	}
	log.Debugf("getmastervalues returned: %v", result)
	return
}

// String returns a string representation of a MasterValueDevice
func (e MasterValueDevice) String() string {
	s := fmt.Sprintf("ID: %s, '%s', Type: %s\n", e.IseID, e.Name, e.DeviceType)
	for _, v := range e.MasterValue {
		s += fmt.Sprintf("  %s=%s\n", v.Name, v.Value)
	}
	return s
}

// String returns a string representation of a MasterValueEntry
func (e MasterValueEntry) String() string {
	return fmt.Sprintf("Name: %s, Value: %s", e.Name, e.Value)
}

// String returns a string representation of a MasterValues
func (e MasterValues) String() string {
	var s string
	for _, v := range e.MasterValueDevices {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}
