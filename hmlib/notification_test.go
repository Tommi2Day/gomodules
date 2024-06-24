package hmlib

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/test"
)

func TestNotification(t *testing.T) {
	test.InitTestDirs()
	httpmock.ActivateNonDefault(httpClient.GetClient())
	defer httpmock.DeactivateAndReset()

	hmURL = MockURL
	hmToken = MockToken
	// mock the response for notifications
	responderURL := hmURL + NotificationsEndpoint
	httpmock.RegisterResponder("GET", responderURL, httpmock.NewStringResponder(200, NotificationsTest))

	stateListURL := hmURL + StateListEndpoint
	httpmock.RegisterResponder("GET", stateListURL, httpmock.NewStringResponder(200, StateListTest))

	t.Run("get_notifications", func(t *testing.T) {
		n, err := GetNotifications()
		assert.NoErrorf(t, err, "GetNotifications should not return an error")
		assert.Equal(t, 2, len(n.Notifications), "GetNotifications should return 1 entry")
		assert.Equal(t, "BidCos-RF.NEQ0117117:0.STICKY_UNREACH", n.Notifications[0].Name, "GetNotifications should return BidCos-RF.NEQ0117117:0.STICKY_UNREACH")
		if len(n.Notifications) > 1 {
			assert.Equal(t, "HmIP-RF.000955699D3D84:0.LOW_BAT", n.Notifications[1].Name, "GetNotifications should return HmIP-RF.000955699D3D84:0.LOW_BAT")
			assert.Equal(t, "LOWBAT", n.Notifications[1].Type, "GetNotifications should return LOWBAT type")
		}
	})
}
