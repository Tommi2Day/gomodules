package hmlib

import (
	"encoding/xml"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DeviceListEndpoint is the endpoint for the device list
const DeviceListEndpoint = "/addons/xmlapi/devicelist.cgi"

// DeviceTypeListEndpoint is the endpoint for the device type list
const DeviceTypeListEndpoint = "/addons/xmlapi/devicetypelist.cgi"

// DeviceAddressMap is a list of devices by name
var DeviceAddressMap = map[string]DeviceListEntry{}

// DeviceIDMap is a list of devices by id
var DeviceIDMap = map[string]DeviceListEntry{}

// DeviceTypes is a list of device types per name
var DeviceTypes = map[string]DeviceTypeEntry{}

// DeviceListResponse is a list of devices returned by API
type DeviceListResponse struct {
	XMLName           xml.Name          `xml:"deviceList"`
	DeviceListEntries []DeviceListEntry `xml:"device"`
}

// DeviceListEntry is a single device
type DeviceListEntry struct {
	XMLName     xml.Name        `xml:"device"`
	Name        string          `xml:"name,attr"`
	Type        string          `xml:"device_type,attr"`
	Address     string          `xml:"address,attr"`
	Interface   string          `xml:"interface,attr"`
	IseID       string          `xml:"ise_id,attr"`
	ReadyConfig string          `xml:"ready_config,attr"`
	Channels    []DeviceChannel `xml:"channel"`
}

// DeviceChannel is a single channel of a device
type DeviceChannel struct {
	XMLName          xml.Name `xml:"channel"`
	Name             string   `xml:"name,attr"`
	Type             string   `xml:"type,attr"`
	Address          string   `xml:"address,attr"`
	IseID            string   `xml:"ise_id,attr"`
	ParentDevice     string   `xml:"parent_device,attr"`
	Index            string   `xml:"index,attr"`
	Direction        string   `xml:"direction,attr"`
	GroupPartner     string   `xml:"group_partner,attr"`
	AesAvailable     string   `xml:"aes_available,attr"`
	TransmissionMode string   `xml:"transmission_mode,attr"`
	Visible          string   `xml:"visible,attr"`
	ReadyConfig      string   `xml:"ready_config,attr"`
	Operate          string   `xml:"operate,attr"`
}

// DeviceTypeListResponse is a list of device types returned by API
type DeviceTypeListResponse struct {
	XMLName               xml.Name          `xml:"deviceTypeList"`
	DeviceTypeListEntries []DeviceTypeEntry `xml:"deviceType"`
}

// DeviceTypeEntry is a single device type
type DeviceTypeEntry struct {
	XMLName       xml.Name         `xml:"deviceType"`
	Name          string           `xml:"name,attr"`
	Description   string           `xml:"description,attr"`
	ThumbnailPath string           `xml:"thumbnailPath,attr"`
	ImagePath     string           `xml:"imagePath,attr"`
	Forms         []DeviceTypeForm `xml:"form"`
}

// DeviceTypeForm is a single form of a device type
type DeviceTypeForm struct {
	XMLName xml.Name `xml:"form"`
	Name    string   `xml:"name,attr"`
	Type    string   `xml:"type,attr"`
}

// GetDeviceList returns the list of devices
func GetDeviceList(deviceIds []string, internal bool) (result DeviceListResponse, err error) {
	result = DeviceListResponse{}
	log.Debug("devicelist called")
	var parameter = map[string]string{}
	if len(deviceIds) > 0 {
		parameter["device_id"] = strings.Join(deviceIds, ",")
	}
	if internal {
		parameter["show_internal"] = "1"
	}
	// reset maps
	AllIds = map[string]IDMapEntry{}
	DeviceAddressMap = map[string]DeviceListEntry{}
	DeviceIDMap = map[string]DeviceListEntry{}
	// query
	err = QueryAPI(DeviceListEndpoint, &result, parameter)
	if err != nil {
		log.Errorf("devicelist returned error: %s", err)
		return
	}
	for _, e := range result.DeviceListEntries {
		DeviceIDMap[e.IseID] = e
		DeviceAddressMap[e.Address] = e
		AllIds[e.IseID] = IDMapEntry{e.IseID, e.Name, "Device", e}
		for _, c := range e.Channels {
			AllIds[c.IseID] = IDMapEntry{c.IseID, c.Name, "Channel", c}
		}
	}
	log.Debugf("devicelist returned %d devices", len(result.DeviceListEntries))
	return
}

// GetDeviceTypeList returns the list of device types
func GetDeviceTypeList() (result DeviceTypeListResponse, err error) {
	result = DeviceTypeListResponse{}
	DeviceTypes = map[string]DeviceTypeEntry{}
	log.Debug("devicetypelist called")
	err = QueryAPI(DeviceTypeListEndpoint, &result, nil)
	if err != nil {
		log.Errorf("devicetypelist returned error: %s", err)
		return
	}
	log.Debugf("devicetypelist returned %d devices", len(result.DeviceTypeListEntries))
	for _, e := range result.DeviceTypeListEntries {
		DeviceTypes[e.Name] = e
	}
	return
}

// String returns a string representation of a device list
func (e DeviceListResponse) String() string {
	var s string
	for _, e := range e.DeviceListEntries {
		s += fmt.Sprintf("%s\n", e)
	}
	return s
}

// String returns a string representation of a device
func (e DeviceListEntry) String() string {
	return fmt.Sprintf("ID:%s, Name: %s, Address: %s, Type: %s", e.IseID, e.Name, e.Address, e.Type)
}

// String returns a string representation of a channel
func (e DeviceChannel) String() string {
	return fmt.Sprintf("  C_ID:%s, Name: %s, Address: %s, Type: %s, Parent: %s ", e.IseID, e.Name, e.Address, e.Type, e.ParentDevice)
}

// String returns a string representation of a device type list
func (e DeviceTypeListResponse) String() string {
	var s string
	for _, e := range e.DeviceTypeListEntries {
		s += fmt.Sprintf("%s\n", e)
	}
	return s
}

// String returns a string representation of a device type
func (e DeviceTypeEntry) String() string {
	return fmt.Sprintf("Name: %s, Description: %s, %d forms", e.Name, e.Description, len(e.Forms))
}

// String returns a string representation of a device type form
func (e DeviceTypeForm) String() string {
	return fmt.Sprintf("Name: %s, Type: %s", e.Name, e.Type)
}
