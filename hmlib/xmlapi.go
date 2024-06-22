package hmlib

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html/charset"
)

var (
	hmToken string
	hmURL   string
)

var httpClient *resty.Client = resty.New()
var debug = false

// QueryAPI function for retrieving via http hmurl/endpoint  and return result as xml
func QueryAPI(endpoint string, result interface{}, parameter map[string]string) (err error) {
	// Create a Resty Client
	if hmToken == "" {
		err = fmt.Errorf("no token set")
		return
	}
	if hmURL == "" {
		err = fmt.Errorf("no hmWarnThreshold server url set")
		return
	}

	if endpoint == "" {
		err = fmt.Errorf("no endpoint set")
		return
	}
	l := log.StandardLogger()
	if debug {
		l.SetLevel(log.DebugLevel)
	} else {
		l.SetLevel(log.ErrorLevel)
	}
	// reset params
	httpClient.QueryParam = url.Values{}
	httpClient.SetHeader("Content-Type", "text/xml")
	httpClient.SetDebug(debug)
	httpClient.SetLogger(l)

	// query params
	qp := fmt.Sprintf("sid=%s", hmToken)
	if len(parameter) > 0 {
		for k, v := range parameter {
			qp += fmt.Sprintf("&%s=%s", k, v)
		}
	}
	callingURL := fmt.Sprintf("%s%s?%s", hmURL, endpoint, qp)
	log.Debugf("query called to %s", callingURL)
	resp, err := httpClient.R().
		EnableTrace().
		Get(callingURL)
	if err != nil {
		err = fmt.Errorf("cannot do request: %s", err)
		return
	}
	if resp.StatusCode() != 200 {
		err = fmt.Errorf("invalid status code: %d", resp.StatusCode())
		return
	}
	data := resp.Body()
	body := string(data)
	log.Debugf("response status: %s", resp.Status())
	log.Debugln("response body:", body)

	if strings.Contains(body, "not_authenticated") {
		err = fmt.Errorf("unauthorized, wrong token?")
		return
	}
	// for iso-8859-1 use decoder.CharsetReader = charset.NewReaderLabel
	// https://github.com/go-resty/resty/issues/481
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&result)
	return
}

// SetHmToken sets the token for the next QueryAPI call
func SetHmToken(token string) {
	hmToken = token
}

// SetHmURL sets the url for the next QueryAPI call
func SetHmURL(url string) {
	hmURL = url
}

// GetHmToken returns the token for the next QueryAPI call
func GetHmToken() string {
	return hmToken
}

// GetHmURL returns the url for the next QueryAPI call
func GetHmURL() string {
	return hmURL
}

// GetHTTPClient returns the http client for the next QueryAPI call
func GetHTTPClient() *resty.Client {
	return httpClient
}

// SetHTTPClient sets the http client for the next QueryAPI call
func SetHTTPClient(c *resty.Client) {
	httpClient = c
}

// SetDebug sets the Logging Level and activates RESTY Debug
func SetDebug(debugFlag bool) {
	debug = debugFlag
}
