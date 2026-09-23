package pwlib

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRDSHost = "mydb.abc123.eu-central-1.rds.amazonaws.com"
const testRDSUser = "dbuser"

// parseRDSToken splits an RDS auth token into host and query values
func parseRDSToken(t *testing.T, token string) (host string, query url.Values) {
	host, rawQuery, ok := strings.Cut(token, "?")
	host = strings.TrimSuffix(host, "/")
	require.Truef(t, ok, "unexpected token format %s", token)
	query, err := url.ParseQuery(rawQuery)
	require.NoError(t, err)
	return host, query
}

func TestRDSAuthToken(t *testing.T) {
	setupAWSTestConfig(t, "http://127.0.0.1:1")
	t.Setenv("RDS_REGION", "")
	SetAWSProfile("plain", "")

	t.Run("Token with port", func(t *testing.T) {
		token, err := GetRDSAuthToken(testRDSHost+":3306", "", testRDSUser)
		require.NoError(t, err)
		host, query := parseRDSToken(t, token)
		assert.Equal(t, testRDSHost+":3306", host)
		assert.Equal(t, "connect", query.Get("Action"))
		assert.Equal(t, testRDSUser, query.Get("DBUser"))
		assert.Contains(t, query.Get("X-Amz-Credential"), "AKIAPLAIN/")
		assert.Contains(t, query.Get("X-Amz-Credential"), "/eu-central-1/rds-db/")
	})
	t.Run("Token with default port", func(t *testing.T) {
		token, err := GetRDSAuthToken(testRDSHost, "", testRDSUser)
		require.NoError(t, err)
		host, _ := parseRDSToken(t, token)
		assert.Equal(t, testRDSHost+":"+RDSDefaultPort, host)
	})
	t.Run("Token with region", func(t *testing.T) {
		token, err := GetRDSAuthToken(testRDSHost, "eu-west-1", testRDSUser)
		require.NoError(t, err)
		_, query := parseRDSToken(t, token)
		assert.Contains(t, query.Get("X-Amz-Credential"), "/eu-west-1/rds-db/")
	})
	t.Run("Missing parameters", func(t *testing.T) {
		_, err := GetRDSAuthToken("", "", testRDSUser)
		require.Error(t, err)
		_, err = GetRDSAuthToken(testRDSHost, "", "")
		require.Error(t, err)
	})
	t.Run("GetPassword with method rds", func(t *testing.T) {
		pc := NewConfig("test_rds", "", "", "", typeRDS)
		require.Equal(t, typeRDS, pc.Method)
		password, err := pc.GetPassword(testRDSHost+":5432", testRDSUser)
		require.NoError(t, err)
		host, query := parseRDSToken(t, password)
		assert.Equal(t, testRDSHost+":5432", host)
		assert.Equal(t, testRDSUser, query.Get("DBUser"))
	})
	t.Run("DecryptFile with method rds", func(t *testing.T) {
		pc := NewConfig("test_rds", "", "", "", typeRDS)
		_, err := pc.DecryptFile()
		require.Error(t, err)
	})
}
