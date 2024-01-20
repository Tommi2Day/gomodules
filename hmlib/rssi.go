package hmlib

import (
	"encoding/xml"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// RssiEndpoint is the endpoint for the device list
const RssiEndpoint = "/addons/xmlapi/rssilist.cgi"

// RssiListResponse is a list of devices returned by API
type RssiListResponse struct {
	XMLName     xml.Name     `xml:"rssiList"`
	RssiDevices []RssiDevice `xml:"rssi"`
}

// RssiDevice returns Rssi values for a single device
type RssiDevice struct {
	XMLName xml.Name `xml:"rssi"`
	Device  string   `xml:"device,attr"`
	Rx      string   `xml:"rx,attr"`
	Tx      string   `xml:"tx,attr"`
}

// RssiDeviceMap is a list of rssi devices by name
var RssiDeviceMap = make(map[string]RssiDevice)

// GetRssiList returns the rssi list of hm devices
func GetRssiList() (result RssiListResponse, err error) {
	log.Debug("getrssi called")
	err = QueryAPI(RssiEndpoint, &result, nil)
	for _, v := range result.RssiDevices {
		RssiDeviceMap[v.Device] = v
	}
	log.Debugf("getRSSI returned %d entries", len(result.RssiDevices))
	return
}

// String returns a string representation of the device list
func (e RssiListResponse) String() string {
	var s string
	for _, v := range e.RssiDevices {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}

// String returns a string representation of the device list
func (e RssiDevice) String() string {
	return fmt.Sprintf("Address:%s rx:%s tx: %s", e.Device, e.Rx, e.Tx)
}
