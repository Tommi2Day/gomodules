package hmlib

import (
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
)

func TestDevice(t *testing.T) {
	var err error
	test.Testinit(t)
	httpmock.ActivateNonDefault(httpClient.GetClient())
	defer httpmock.DeactivateAndReset()
	hmURL = MockURL
	hmToken = MockToken
	var deviceList DeviceListResponse
	t.Run("device list", func(t *testing.T) {
		response := DeviceListTest
		responder := httpmock.NewStringResponder(200, response)
		fakeURL := hmURL + DeviceListEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)

		deviceList, err = GetDeviceList("", false)
		assert.NoErrorf(t, err, "GetDeviceList should not return an error:%s", err)
		require.Greater(t, len(deviceList.DeviceListEntries), 0, "GetDeviceList should return entries")
		assert.Equal(t, 1, len(DeviceAddressMap), "GetDeviceList should return 1 entry")
		assert.Equal(t, 1, len(DeviceIDMap), "GetDeviceList should return 1 entry")
		d, ok := DeviceAddressMap["000955699D3D84"]
		assert.True(t, ok, "GetDeviceList should contain 000955699D3D84")
		assert.Equal(t, "Bewegungsmelder Garage", d.Name, "GetDeviceList should contain Bewegungsmelder Garage")
		d, ok = DeviceIDMap["4740"]
		assert.True(t, ok, "GetDeviceList should contain 4740")
		assert.Equal(t, "Bewegungsmelder Garage", d.Name, "GetDeviceList should contain Bewegungsmelder Garage")
		assert.Equal(t, "HmIP-SMO", d.Type, "GetDeviceList should contain HmIP-SMO")
		assert.Equal(t, len(d.Channels), 4, "GetDeviceList should contain 4 channels")
		t.Log(deviceList.String())
	})
	t.Run("device list empty", func(t *testing.T) {
		response := DeviceListEmptyTest
		responder := httpmock.NewStringResponder(200, response)
		fakeURL := hmURL + DeviceListEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)
		deviceList, err = GetDeviceList("", true)
		assert.NoErrorf(t, err, "GetDeviceList should not return an error:%s", err)
		require.Equal(t, len(deviceList.DeviceListEntries), 0, "GetDeviceList should return entries")
		assert.Equal(t, 0, len(DeviceAddressMap), "GetDeviceList should return 0 entry")
		assert.Equal(t, 0, len(DeviceIDMap), "GetDeviceList should return 0 entry")
	})
	t.Run("device types list", func(t *testing.T) {
		response := DeviceTypeListTest
		responder := httpmock.NewStringResponder(200, response)
		fakeURL := hmURL + DeviceTypeListEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)
		deviceTypes, err := GetDeviceTypeList()
		assert.NoErrorf(t, err, "GetDeviceTypes should not return an error:%s", err)
		assert.Equal(t, 5, len(deviceTypes.DeviceTypeListEntries), "GetDeviceTypes should return 5 entries")
		if len(deviceTypes.DeviceTypeListEntries) > 0 {
			e := deviceTypes.DeviceTypeListEntries[0]
			assert.Equal(t, "HM-RC-Sec4-3", e.Name, "GetDeviceType[0] should return HM-RC-Sec4-3")
			assert.Equal(t, "HM-RC-4", e.Description, "GetDeviceType[0] should return HM-RC-4")
			l := len(e.Forms)
			assert.Greater(t, l, 1, "GetDeviceType[0] should return more 1 form")
			if l > 0 {
				t.Logf("Form[0]:%s", e.Forms[0].String())
			}
		}
		t.Log(deviceTypes.String())
	})
	t.Run("device types list not autenticated", func(t *testing.T) {
		fakeURL := "http://localhost:80" + DeviceTypeListEndpoint
		sid := "xxx"
		SetHmToken(sid)
		SetHmURL("http://localhost:80")
		q := url.Values{
			"sid": []string{sid},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", fakeURL, q,
			httpmock.NewStringResponder(200, DeviceTypeListNotAuthTest))
		deviceTypes, err := GetDeviceTypeList()
		assert.Errorf(t, err, "GetDeviceTypes should return an error:%s", err)
		assert.Equal(t, 0, len(deviceTypes.DeviceTypeListEntries), "GetDeviceTypes should return 0 entries")
	})
}
