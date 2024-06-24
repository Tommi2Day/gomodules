package hmlib

import (
	"encoding/xml"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/test"
)

const mockTestEndpoint = "/addons/xmlapi/test.cgi"
const mockTestResponse = `<test><status><message>Working</message><code>200</code></status></test>`
const mockInvalidEndpoint = "/addons/xmlapi/invalid.cgi"

type mockTest struct {
	XMLName xml.Name   `xml:"test"`
	Status  mockStatus `xml:"status"`
}
type mockStatus struct {
	XMLName xml.Name `xml:"status"`
	Message string   `xml:"message"`
	Code    string   `xml:"code"`
}

func TestQueryAPI(t *testing.T) {
	var err error
	var result mockTest
	test.InitTestDirs()

	httpmock.ActivateNonDefault(httpClient.GetClient())
	defer httpmock.DeactivateAndReset()
	t.Run("QueryAPI with empty token", func(t *testing.T) {
		hmToken = ""
		err = QueryAPI(mockTestEndpoint, &result, nil)
		t.Log(err)
		assert.Error(t, err, "QueryAPI should return an error")
	})

	hmToken = MockToken
	t.Run("QueryAPI with empty endpoint", func(t *testing.T) {
		err = QueryAPI("", &result, nil)
		t.Log(err)
		assert.Error(t, err, "QueryAPI should return an error")
	})
	t.Run("QueryAPI with empty url", func(t *testing.T) {
		hmURL = ""
		err = QueryAPI(mockTestEndpoint, &result, nil)
		t.Log(err)
		assert.Error(t, err, "QueryAPI should return an error")
	})
	hmURL = MockURL
	t.Run("QueryAPI with valid endpoint", func(t *testing.T) {
		response := mockTestResponse
		responder := httpmock.NewStringResponder(200, response)
		fakeURL := MockURL + mockTestEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)

		err = QueryAPI(mockTestEndpoint, &result, nil)
		t.Log(result)
		assert.NoErrorf(t, err, "QueryAPI should not return an error:%s", err)
		assert.Equal(t, "Working", result.Status.Message, "QueryAPI should return a valid result")
	})
	t.Run("QueryAPI with invalid endpoint", func(t *testing.T) {
		response := ""
		responder := httpmock.NewStringResponder(404, response)
		fakeURL := MockURL + mockInvalidEndpoint
		httpmock.RegisterResponder("GET", fakeURL, responder)
		err = QueryAPI(mockInvalidEndpoint, &result, nil)
		t.Log(err)
		assert.Error(t, err, "QueryAPI should return an error")
		assert.Contains(t, err.Error(), "invalid status code: 404", "QueryAPI should return an error")
	})
}
