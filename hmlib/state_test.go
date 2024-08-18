package hmlib

import (
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
)

func TestState(t *testing.T) {
	test.InitTestDirs()
	httpmock.ActivateNonDefault(httpClient.GetClient())
	hmURL = MockURL
	hmToken = MockToken
	defer httpmock.DeactivateAndReset()
	stateListURL := hmURL + StateListEndpoint
	httpmock.RegisterResponder("GET", stateListURL, httpmock.NewStringResponder(200, StateListTest))

	stateURL := hmURL + StateEndpoint
	changeURL := hmURL + StateChangeEndpoint
	httpmock.RegisterResponder("GET", stateURL, httpmock.NewStringResponder(200, StateTest))
	// mock the response for state
	var stateList StateListResponse
	var err error
	t.Run("statelist", func(t *testing.T) {
		stateList, err = GetStateList()
		require.NoErrorf(t, err, "GetStateList should not return an error:%s", err)
		sl := len(stateList.StateDevices)
		assert.Equal(t, len(StateList.StateDevices), sl, "global StateList should equal GetStateList devices")
		assert.Equal(t, 2, sl, "GetStateList should return 2 devices")
		l := len(AllIDs)
		assert.Equal(t, 44, l, "AllIDs should return 44 entries")
		e, ok := AllIDs["4740"]
		assert.True(t, ok, "AllIDs should contain 4740")
		if ok {
			assert.Equal(t, "Bewegungsmelder Garage", e.Name, "AllIDs should contain Bewegungsmelder Garage")
			assert.Equal(t, "Device", e.EntryType, "ID 4740 should be a device")
			assert.IsType(t, StateDevice{}, e.Entry, "ID 4740 should be a device")
			c := e.Entry.(StateDevice).Channels
			assert.Equal(t, 4, len(c), "ID 4740 should contain 4 channels")
			if len(c) > 0 {
				assert.Contains(t, c[0].Name, "Bewegungsmelder Garage:0", "ID 4740 should contain channel Bewegungsmelder Garage:0")
			}
		}
	})
	t.Run("single device state", func(t *testing.T) {
		var s StateDeviceResponse
		s, err = GetStateByDeviceID("4740")
		require.NoErrorf(t, err, "GetStateBy should not return an error:%s", err)
		assert.Equal(t, 1, len(s.StateDevices), "GetStateByDeviceID should return 1 device")
		if len(s.StateDevices) > 0 {
			assert.Containsf(t, s.StateDevices[0].Name, "Bewegungsmelder Garage", "GetStateByDeviceID should return Bewegungsmelder Garage")
			assert.Equal(t, 4, len(s.StateDevices[0].Channels), "GetStateByDeviceID should return 1 channel")
		}
		t.Log(s.String())
	})
	t.Run("single channel state", func(t *testing.T) {
		var s StateDeviceResponse
		s, err = GetStateByChannelID("4741")
		require.NoErrorf(t, err, "GetStateBy should not return an error:%s", err)
		assert.Equal(t, 1, len(s.StateDevices), "GetStateByDeviceID should return 1 device")
		if len(s.StateDevices) > 0 {
			c := s.StateDevices[0].Channels
			assert.Equal(t, 4, len(c), "GetStateByChannelID should return 1 channel")
			assert.Containsf(t, c[0].Name, "Bewegungsmelder Garage:0", "GetStateByChannelID should return Bewegungsmelder Garage:0")
			assert.Equal(t, 10, len(c[0].Datapoints), "GetStateByChannelID should return 1 Datapoints")
		}
		t.Log(s.String())
	})
	t.Run("single datapoint state", func(t *testing.T) {
		var s StateDatapointResponse
		queryDP := url.Values{
			"datapoint_id": []string{"4748"},
			"sid":          []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", stateURL, queryDP,
			httpmock.NewStringResponder(200, StateDP4748))
		s, err := GetStateByDataPointID("4748")
		require.NoErrorf(t, err, "GetStateBy should not return an error:%s", err)
		assert.Equal(t, 1, len(s.StateDatapoints), "GetStateByDatapointID should return 1 datapoint")
		if len(s.StateDatapoints) > 0 {
			assert.Equal(t, s.StateDatapoints[0].IseID, "4748", "GetStateByDatapointID should return ID 4748")
			assert.Equal(t, s.StateDatapoints[0].Value, "false", "GetStateByDatapointID should return value false")
		}
		t.Log(s.String())
	})
	t.Run("State Empty", func(t *testing.T) {
		var s StateDatapointResponse
		queryStateEmpty := url.Values{
			"datapoint_id": []string{"9999"},
			"sid":          []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", stateURL, queryStateEmpty,
			httpmock.NewStringResponder(200, StateEmptyTest))

		s, err = GetStateByDataPointID("9999")
		require.NoErrorf(t, err, "GetStateBy should not return an error:%s", err)
		assert.Equal(t, 0, len(s.StateDatapoints), "GetStateByDatapointID should return 0 datapoint")
		t.Log(s.String())
	})
	t.Run("state change", func(t *testing.T) {
		var r StateChangeResponse

		queryChange := url.Values{
			"ise_id":    []string{"4740"},
			"new_value": []string{"11"},
			"sid":       []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", changeURL, queryChange,
			httpmock.NewStringResponder(200, StateChangeTest))
		r, err = ChangeState("4740", "11")
		require.NoErrorf(t, err, "ChangeState should not return an error")
		l := len(r.Changes)
		assert.Equal(t, 1, l, "GetMasterValues should return 1 devices")
		if l == 1 {
			assert.True(t, r.Changes[0].Success, "ChangeState should return the new value")
		} else {
			t.Errorf("ChangeState should return the changed entry")
		}
		t.Log(r.String())
	})
	t.Run("one state change not found, space in list", func(t *testing.T) {
		var r StateChangeResponse
		queryChange := url.Values{
			"ise_id":    []string{"474,4740"},
			"new_value": []string{"11"},
			"sid":       []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", changeURL, queryChange,
			httpmock.NewStringResponder(200, StateChangeNotFoundTest))
		r, err = ChangeState("474, 4740", "11")
		require.Errorf(t, err, "ChangeState should return an error")
		l := len(r.NotFound)
		assert.Equal(t, 1, l, "ChangeState should return 1 id not found")
		l = len(r.Changes)
		assert.Equal(t, 1, l, "ChangeState should return 1 id success ")
		if l == 1 {
			assert.Equal(t, "4740", r.Changes[0].IseID, "ChangeState should return ID 4740")
			assert.True(t, r.Changes[0].Success, "ChangeState should return true for ID 4740")
		}
		t.Log(r.String())
	})
	t.Run("state change empty", func(t *testing.T) {
		var r StateChangeResponse
		queryChange := url.Values{
			"device_id": []string{"4740"},
			"new_value": []string{"11"},
			"sid":       []string{hmToken},
		}
		p := map[string]string{
			"device_id": "4740",
			"new_value": "11",
			"sid":       hmToken,
		}
		httpmock.RegisterResponderWithQuery(
			"GET", changeURL, queryChange,
			httpmock.NewStringResponder(200, StateChangeEmptyTest))
		err = QueryAPI(changeURL, &r, p)
		require.Errorf(t, err, "ChangeState should return an error")
		t.Log(r.String())
	})
	t.Run("GetChannelOfDatapoint", func(t *testing.T) {
		var channelID string
		channelID, err = GetChannelOfDatapoint("4748")
		require.NoErrorf(t, err, "GetChannelOfDatapoint should not return an error:%s", err)
		assert.Equal(t, "4741", channelID, "GetChannelOfDatapoint should return channel 4741")
	})
	t.Run("GetChannelOfDatapoint not found", func(t *testing.T) {
		var channelID string
		channelID, err = GetChannelOfDatapoint("9999")
		require.Errorf(t, err, "GetChannelOfDatapoint should return an error")
		assert.Equal(t, "", channelID, "GetChannelOfDatapoint should return empty channel")
	})
	t.Run("GetChannelOfDatapoint not a datapoint", func(t *testing.T) {
		var channelID string
		channelID, err = GetChannelOfDatapoint("4740")
		require.Errorf(t, err, "GetChannelOfDatapoint should return an error")
		assert.Equal(t, "", channelID, "GetChannelOfDatapoint should return empty channel")
	})
	t.Run("GetChannelOfDatapoint empty", func(t *testing.T) {
		var channelID string
		channelID, err = GetChannelOfDatapoint("")
		require.Errorf(t, err, "GetChannelOfDatapoint should return an error")
		assert.Equal(t, "", channelID, "GetChannelOfDatapoint should return empty channel")
	})
	t.Run("GetDeviceOfChannel", func(t *testing.T) {
		var deviceID string
		deviceID, err = GetDeviceOfChannel("4741")
		require.NoErrorf(t, err, "GetDeviceOfChannel should not return an error:%s", err)
		assert.Equal(t, "4740", deviceID, "GetDeviceOfChannel should return device 4740")
	})
	t.Run("GetDeviceOfChannel not found", func(t *testing.T) {
		var deviceID string
		deviceID, err = GetDeviceOfChannel("9999")
		require.Errorf(t, err, "GetDeviceOfChannel should return an error")
		assert.Equal(t, "", deviceID, "GetDeviceOfChannel should return empty device")
	})
	t.Run("GetDeviceOfChannel not a channel", func(t *testing.T) {
		var deviceID string
		deviceID, err = GetDeviceOfChannel("4740")
		require.Errorf(t, err, "GetDeviceOfChannel should return an error")
		assert.Equal(t, "", deviceID, "GetDeviceOfChannel should return empty device")
	})
	t.Run("GetDeviceOfChannel empty", func(t *testing.T) {
		var deviceID string
		deviceID, err = GetDeviceOfChannel("")
		require.Errorf(t, err, "GetDeviceOfChannel should return an error")
		assert.Equal(t, "", deviceID, "GetDeviceOfChannel should return empty device")
	})
}
