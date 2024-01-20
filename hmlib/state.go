package hmlib

import (
	"encoding/xml"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// StateEndpoint is the endpoint to retrieve the state for a given list of devices
const StateEndpoint = "/addons/xmlapi/state.cgi"

// StateChangeEndpoint is the endpoint to change a state value
const StateChangeEndpoint = "/addons/xmlapi/statechange.cgi"

// StateListEndpoint is the endpoint to retrieve the state for all devices
const StateListEndpoint = "/addons/xmlapi/statelist.cgi"

// StateListResponse is a list of devices returned by API
type StateListResponse struct {
	XMLName      xml.Name      `xml:"stateList"`
	StateDevices []StateDevice `xml:"device"`
}

// StateDeviceResponse is a list of devices returned by API
type StateDeviceResponse struct {
	XMLName      xml.Name      `xml:"state"`
	StateDevices []StateDevice `xml:"device"`
}

// StateDatapointResponse is a list of datapoints returned by API
type StateDatapointResponse struct {
	XMLName         xml.Name    `xml:"state"`
	StateDatapoints []Datapoint `xml:"datapoint"`
}

// StateDevice returns the state of a single device
type StateDevice struct {
	XMLName       xml.Name       `xml:"device"`
	Name          string         `xml:"name,attr"`
	IseID         string         `xml:"ise_id,attr"`
	Unreach       string         `xml:"unreach,attr"`
	ConfigPending string         `xml:"config_pending,attr"`
	Channels      []StateChannel `xml:"channel"`
}

// StateChannel returns the state of a single channel
type StateChannel struct {
	XMLName          xml.Name    `xml:"channel"`
	Name             string      `xml:"name,attr"`
	IseID            string      `xml:"ise_id,attr"`
	LastDPActionTime string      `xml:"lastdpactiontime,attr"`
	Datapoints       []Datapoint `xml:"datapoint"`
}

// Datapoint returns the state of a single datapoint
type Datapoint struct {
	XMLName       xml.Name `xml:"datapoint"`
	Name          string   `xml:"name,attr"`
	IseID         string   `xml:"ise_id,attr"`
	Value         string   `xml:"value,attr"`
	ValueType     string   `xml:"valuetype,attr"`
	ValueUnit     string   `xml:"valueunit,attr"`
	Timestamp     string   `xml:"timestamp,attr"`
	LastTimestamp string   `xml:"lasttimestamp,attr"`
}

// StateChangeResponse is the result of a statechange.cgi call
type StateChangeResponse struct {
	XMLName  xml.Name       `xml:"result"`
	NotFound []bool         `xml:"not_found"`
	Changes  []ChangeResult `xml:"changed"`
}

// ChangeResult is the result of a single change
type ChangeResult struct {
	XMLName  xml.Name `xml:"changed"`
	IseID    string   `xml:"id,attr"`
	NewValue string   `xml:"new_value,attr"`
	Success  bool     `xml:"success,attr"`
}

// StateList is the result of a statelist.cgi call
var StateList = StateListResponse{}

// GetStateList returns the state of all devices
func GetStateList() (stateList StateListResponse, err error) {
	stateList = StateListResponse{}
	log.Debug("getstatelist called")
	err = QueryAPI(StateListEndpoint, &stateList, nil)
	if err != nil {
		log.Errorf("getstatelist returned error: %s", err)
		return
	}
	for _, e := range stateList.StateDevices {
		AllIds[e.IseID] = IDMapEntry{e.IseID, e.Name, "Device", e}
		NameIDMap[e.Name] = e.IseID
		for _, c := range e.Channels {
			AllIds[c.IseID] = IDMapEntry{c.IseID, c.Name, "Channel", c}
			NameIDMap[c.Name] = c.IseID
			for _, d := range c.Datapoints {
				AllIds[d.IseID] = IDMapEntry{d.IseID, d.Name, "Datapoint", d}
				NameIDMap[d.Name] = d.IseID
			}
		}
	}
	log.Debugf("getstateList returned %d IDs", len(AllIds))
	StateList = stateList
	return
}

// GetStateByDeviceID returns the state of the given devices
func GetStateByDeviceID(ids []string) (result StateDeviceResponse, err error) {
	log.Debug("getstatebyid called")
	if len(ids) == 0 {
		err = fmt.Errorf("no ids given")
		return
	}
	parameter := map[string]string{"device_id": strings.Join(ids, ",")}
	err = QueryAPI(StateEndpoint, &result, parameter)
	log.Debugf("getstate returned: %v", result)
	return
}

// GetStateByChannelID returns the state of the given channels
func GetStateByChannelID(ids []string) (result StateDeviceResponse, err error) {
	log.Debug("getstatebychannelid called")
	if len(ids) == 0 {
		err = fmt.Errorf("no ids given")
		return
	}
	parameter := map[string]string{"channel_id": strings.Join(ids, ",")}
	err = QueryAPI(StateEndpoint, &result, parameter)
	log.Debugf("getstate returned: %v", result)
	return
}

// GetStateByDataPointID returns the state of the given datapoints
func GetStateByDataPointID(ids []string) (result StateDatapointResponse, err error) {
	log.Debug("getstatebydpid called")
	if len(ids) == 0 {
		err = fmt.Errorf("no ids given")
		return
	}
	parameter := map[string]string{"datapoint_id": strings.Join(ids, ",")}
	err = QueryAPI(StateEndpoint, &result, parameter)
	log.Debugf("getstate returned: %v", result)
	return
}

// ChangeState changes the state of the given id
func ChangeState(ids []string, values []string) (result StateChangeResponse, err error) {
	result = StateChangeResponse{}
	log.Debug("changestatebyid called")
	if len(ids) == 0 {
		err = fmt.Errorf("no ids given")
		return
	}
	if len(values) == 0 {
		err = fmt.Errorf("no values given")
		return
	}
	parameter := map[string]string{"ise_id": strings.Join(ids, ",")}
	parameter["new_value"] = strings.Join(values, ",")
	err = QueryAPI(StateChangeEndpoint, &result, parameter)
	if err != nil {
		err = fmt.Errorf("value query Error id %v", err)
		return
	}

	if len(result.Changes) == 0 && len(result.NotFound) == 0 {
		err = fmt.Errorf("no changes, wrong parameter?")
		return
	}
	l := len(result.NotFound)
	if l > 0 {
		err = fmt.Errorf("%d ids not found", l)
		return
	}

	log.Debugf("changestate returned: %v", result)
	return
}

// String returns a string representation of a StateListResponse
func (e StateDevice) String() string {
	return fmt.Sprintf("ID:%s, Name: %s", e.IseID, e.Name)
}

// String returns a string representation of a StateChannel
func (e StateChannel) String() string {
	return fmt.Sprintf("ID:%s, Name: %s", e.IseID, e.Name)
}

// String returns a string representation of a Datapoint
func (e Datapoint) String() string {
	return fmt.Sprintf("ID:%s, Name: %s, Value: %s%s, Type: %s Last: %s", e.IseID, e.Name, e.Value, e.ValueUnit, e.ValueType, e.LastTimestamp)
}

// String returns a string representation of a StateDeviceResponse
func (e StateDeviceResponse) String() string {
	var s string
	for _, v := range e.StateDevices {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}

// String returns a string representation of a StateDatapointResponse
func (e StateDatapointResponse) String() string {
	var s string
	for _, v := range e.StateDatapoints {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}

func (e StateChangeResponse) String() string {
	var s string
	for _, v := range e.Changes {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}

// String returns a string representation of a ChangeResult
func (e ChangeResult) String() string {
	return fmt.Sprintf("ID:%s, NewValue: %s, Success: %v", e.IseID, e.NewValue, e.Success)
}
