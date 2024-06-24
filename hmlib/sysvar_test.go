package hmlib

import (
	"net/url"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/test"
)

func TestSysvar(t *testing.T) {
	var err error
	test.InitTestDirs()
	httpmock.ActivateNonDefault(httpClient.GetClient())
	defer httpmock.DeactivateAndReset()
	hmURL = MockURL
	hmToken = MockToken

	t.Run("sysvar list", func(t *testing.T) {
		response := SysVarListTest
		responder := httpmock.NewStringResponder(200, response)
		fakeURL := hmURL + SysVarListEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)

		err = GetSysvarList(true)
		assert.NoErrorf(t, err, "GetSysvarList should not return an error:%s", err)
		assert.Equal(t, 7, len(SysVarIDMap), "GetSysvarList should return 7 entries")
		s, ok := AllIDs["8254"]
		assert.True(t, ok, "GetDeviceList should contain 4740")
		if ok && s.EntryType == "systemVariable" {
			assert.Equal(t, "Sysvar", s.EntryType, "GetDeviceList should contain 6.000000")
			assert.Equal(t, "DutyCycle-LGW", s.Name, "GetSysvarList 8264 Name should contain DutyCycle-LGW")
		}
		t.Log(s)
	})

	t.Run("sysvar list empty", func(t *testing.T) {
		SysVarIDMap = map[string]SysVarEntry{}
		fakeURL := hmURL + SysVarListEndpoint
		queryVar := url.Values{
			"text": []string{"true"},
			"sid":  []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", fakeURL, queryVar,
			httpmock.NewStringResponder(200, SysVarEmptyTest))
		err = GetSysvarList(true)
		assert.NoErrorf(t, err, "GetSysvarList should not return an error:%s", err)
		assert.Equal(t, 0, len(SysVarIDMap), "GetSysvarList should return 0 entry")
	})

	t.Run("sysvar empty", func(t *testing.T) {
		SysVarIDMap = map[string]SysVarEntry{}
		fakeURL := hmURL + SysVarEndpoint
		// mock the response for state
		queryVar := url.Values{
			"ise_id": []string{"4711"},
			"sid":    []string{hmToken},
		}
		httpmock.RegisterResponderWithQuery(
			"GET", fakeURL, queryVar,
			httpmock.NewStringResponder(200, SysVarEmptyTest))
		l, err := GetSysvar("4711", false)
		assert.NoErrorf(t, err, "GetSysvar should not return an error:%s", err)
		assert.NotNil(t, l, "GetSysvar should return a list")
		assert.Equal(t, 0, len(l.SysvarEntry), "GetSysvar should return 0 entry")
	})
}
