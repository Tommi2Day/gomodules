package hmlib

import (
	"net/url"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
)

func TestValue(t *testing.T) {
	test.InitTestDirs()
	httpmock.ActivateNonDefault(httpClient.GetClient())
	hmURL = MockURL
	hmToken = MockToken
	defer httpmock.DeactivateAndReset()
	valueURL := hmURL + MasterValueEndpoint
	changeURL := hmURL + MasterValueChangeEndpoint
	t.Run("mastervalue", func(t *testing.T) {
		var v MasterValues
		var err error
		rn := "ARR_TIMEOUT,LOW_BAT_LIMIT"
		devices := "4740,4741"
		queryValue := url.Values{
			"device_id":       []string{devices},
			"requested_names": []string{rn},
			"sid":             []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", valueURL, queryValue,
			httpmock.NewStringResponder(200, MasterValueTest))
		v, err = GetMasterValues(devices, rn)
		require.NoErrorf(t, err, "GetValueList should not return an error:%s", err)
		l := len(v.MasterValueDevices)
		assert.Equal(t, 2, l, "GetMasterValues should return 2 devices")
		if l > 0 {
			assert.Equal(t, 12, len(v.MasterValueDevices[0].MasterValue), "GetMasterValues should return 2 values")
		}
		t.Log(v.String())
	})
	t.Run("mastervalue error", func(t *testing.T) {
		var v MasterValues
		var err error
		rn := "ARR_TIMEOUT,CYCLIC_BIDI_INFO_MSG_DISCARD_FACTOR"
		queryValueError := url.Values{
			"device_id":       []string{"2850"},
			"requested_names": []string{rn},
			"sid":             []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", valueURL, queryValueError,
			httpmock.NewStringResponder(200, MasterValueErrorTest))
		v, err = GetMasterValues("2850", rn)
		require.Errorf(t, err, "GetValueList should return an error")
		l := len(v.MasterValueDevices)
		assert.Equal(t, 1, l, "GetMasterValues should return 1 devices")
		if l == 1 {
			c := strings.TrimSpace(v.MasterValueDevices[0].Content)
			assert.Equal(t, "DEVICE NOT FOUND", c, "GetMasterValues should return Error Message")
		}
		t.Log(v.String())
	})
	t.Run("mastervalue change", func(t *testing.T) {
		var v MasterValues
		var err error
		queryChange := url.Values{
			"device_id": []string{"4740"},
			"name":      []string{"ARR_TIMEOUT"},
			"value":     []string{"11"},
			"sid":       []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", changeURL, queryChange,
			httpmock.NewStringResponder(200, MasterValueChangeTest))
		v, err = ChangeMasterValues("4740", "ARR_TIMEOUT", "11")
		require.NoErrorf(t, err, "ChangeMasterValues should not return an error")
		l := len(v.MasterValueDevices)
		assert.Equal(t, 1, l, "GetMasterValues should return 1 devices")
		if l == 1 && len(v.MasterValueDevices[0].MasterValue) > 0 {
			c := strings.TrimSpace(v.MasterValueDevices[0].MasterValue[0].Value)
			assert.Equal(t, "11", c, "ChangeMasterValues should return the new value")
		} else {
			t.Errorf("ChangeMasterValues should return the changed entry")
		}
		t.Log(v.String())
	})
}
