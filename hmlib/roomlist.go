package hmlib

import (
	"encoding/xml"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// RoomListEndpoint is the endpoint for the device list
const RoomListEndpoint = "/addons/xmlapi/roomlist.cgi"

// RoomListResponse is a list of devices returned by API
type RoomListResponse struct {
	XMLName xml.Name `xml:"roomList"`
	Rooms   []Room   `xml:"room"`
}

// Room is a single room in RoomListResponse
type Room struct {
	XMLName  xml.Name      `xml:"room"`
	Name     string        `xml:"name,attr"`
	IseID    string        `xml:"ise_id,attr"`
	Channels []RoomChannel `xml:"channel"`
}

// RoomChannel is a single channel in Room
type RoomChannel struct {
	XMLName xml.Name `xml:"channel"`
	IseID   string   `xml:"ise_id,attr"`
}

// RoomMap is a list of rooms by name
var RoomMap = make(map[string]Room)

// GetRoomList returns the list of rooms
func GetRoomList() (result RoomListResponse, err error) {
	log.Debug("getrssi called")
	err = QueryAPI(RoomListEndpoint, &result, nil)
	for _, v := range result.Rooms {
		RoomMap[v.Name] = v
	}
	log.Debugf("getRoomList returned %d rooms", len(result.Rooms))
	return
}

// String returns a string representation of the device list
func (e RoomListResponse) String() string {
	var s string
	for _, v := range e.Rooms {
		s += fmt.Sprintf("%s\n", v)
	}
	return s
}

// String returns a string representation of the device list
func (e Room) String() string {
	return fmt.Sprintf("Name:%s Channels:%d ", e.Name, len(e.Channels))
}
