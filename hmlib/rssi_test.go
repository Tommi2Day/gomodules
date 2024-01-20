package hmlib

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/test"
)

func TestRssi(t *testing.T) {
	test.Testinit(t)
	httpmock.ActivateNonDefault(httpClient.GetClient())
	response := RssiTest
	responder := httpmock.NewStringResponder(200, response)
	fakeURL := hmURL + RssiEndpoint
	httpmock.RegisterResponder("GET", fakeURL, responder)
	defer httpmock.DeactivateAndReset()

	t.Run("Rssi func", func(t *testing.T) {
		actual, err := GetRssiList()
		t.Logf(actual.String())
		assert.NoErrorf(t, err, "GetRssiList should not return an error:%s", err)
		assert.Equal(t, 4, len(actual.RssiDevices), "GetRssiList should return 4 entries")
		_, ok := RssiDeviceMap["MEQ0481419"]
		assert.True(t, ok, "GetRssiList should contain MEQ0481419")
	})
}
