package hmlib

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/test"
)

func TestRoomlist(t *testing.T) {
	test.Testinit(t)
	httpmock.ActivateNonDefault(httpClient.GetClient())
	response := RoomListTest
	responder := httpmock.NewStringResponder(200, response)
	fakeURL := hmURL + RoomListEndpoint
	httpmock.RegisterResponder("GET", fakeURL, responder)
	defer httpmock.DeactivateAndReset()

	t.Run("RoomListResponse", func(t *testing.T) {
		actual, err := GetRoomList()
		t.Logf(actual.String())
		assert.NoErrorf(t, err, "GetRoomList should not return an error:%s", err)
		assert.Equal(t, 2, len(actual.Rooms), "GetRoomList should return 2 entries")
		idl := map[string]string{}
		for _, r := range actual.Rooms {
			rn := r.Name
			cl := r.Channels
			for _, c := range cl {
				idl[c.IseID] = rn
			}
		}
		n, ok := idl["3076"]
		assert.True(t, ok, "GetRoomList should contain channel 3076")
		assert.Equal(t, "Bad", n, "GetRoomList should contain channel 3076 in Bad")
	})
}
