package hmlib

import (
	"encoding/xml"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// NotificationsEndpoint is the endpoint for the notification list
const NotificationsEndpoint = "/config/xmlapi/systemNotification.cgi"

// SystemNotificationResponse is a list of notifications returned by API
type SystemNotificationResponse struct {
	XMLName       xml.Name       `xml:"systemNotification"`
	Notifications []Notification `xml:"notification"`
}

// NotificationDetail is a aggregated single notification
type NotificationDetail struct {
	System  string
	Address string
	Type    string
	Since   time.Time
	Name    string
}

// Notification is a single notification
type Notification struct {
	XMLName   xml.Name `xml:"notification"`
	Name      string   `xml:"name,attr"`
	Type      string   `xml:"type,attr"`
	Timestamp string   `xml:"timestamp,attr"`
	IseID     string   `xml:"ise_id,attr"`
}

// String returns a string representation of the notification
func (e Notification) String() string {
	return fmt.Sprintf("ID:%s, %s", e.IseID, e.Name)
}

// GetNotifications returns the list of notifications
func GetNotifications() (result SystemNotificationResponse, err error) {
	log.Debug("getnotifications called")
	err = QueryAPI(NotificationsEndpoint, &result, nil)
	log.Debugf("notifications returned: %v", result)
	return
}
