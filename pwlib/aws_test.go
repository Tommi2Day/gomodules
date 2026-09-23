package pwlib

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMFAToken = "123456"
const testMFASerial = "arn:aws:iam::123456789012:mfa/tester"

const stsCredentialsXML = `<Credentials>
      <AccessKeyId>%s</AccessKeyId>
      <SecretAccessKey>stsSecret</SecretAccessKey>
      <SessionToken>stsSession</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>`

// fakeSTS mocks STS AssumeRole and GetSessionToken and records the received requests
type fakeSTS struct {
	mu       sync.Mutex
	requests []map[string]string
}

func (f *fakeSTS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req := map[string]string{
		"Action":       r.Form.Get("Action"),
		"TokenCode":    r.Form.Get("TokenCode"),
		"SerialNumber": r.Form.Get("SerialNumber"),
		"RoleArn":      r.Form.Get("RoleArn"),
	}
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	exp := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	action := req["Action"]
	w.Header().Set("Content-Type", "text/xml")
	switch action {
	case "AssumeRole":
		_, _ = fmt.Fprintf(w, `<AssumeRoleResponse><AssumeRoleResult>`+stsCredentialsXML+
			`<AssumedRoleUser><Arn>arn:aws:sts::123456789012:assumed-role/test/s</Arn><AssumedRoleId>ARO:s</AssumedRoleId></AssumedRoleUser>`+
			`</AssumeRoleResult></AssumeRoleResponse>`, "ASIAROLE", exp)
	case "GetSessionToken":
		_, _ = fmt.Fprintf(w, `<GetSessionTokenResponse><GetSessionTokenResult>`+stsCredentialsXML+
			`</GetSessionTokenResult></GetSessionTokenResponse>`, "ASIASESSION", exp)
	default:
		http.Error(w, "unsupported action "+action, http.StatusBadRequest)
	}
}

func (f *fakeSTS) lastRequest() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return nil
	}
	return f.requests[len(f.requests)-1]
}

// setupAWSTestConfig writes shared config and credentials files and points the SDK to them
func setupAWSTestConfig(t *testing.T, stsURL string) {
	dir := t.TempDir()
	cfgFile := path.Join(dir, "config")
	credFile := path.Join(dir, "credentials")
	cfgContent := fmt.Sprintf(`[profile plain]
region = eu-central-1

[profile mfauser]
region = eu-central-1
mfa_serial = %[1]s

[profile mfarole]
region = eu-central-1
role_arn = arn:aws:iam::123456789012:role/test
source_profile = mfauser
mfa_serial = %[1]s
`, testMFASerial)
	//nolint:gosec // fake test credentials
	credContent := `[plain]
aws_access_key_id = AKIAPLAIN
aws_secret_access_key = plainSecret

[mfauser]
aws_access_key_id = AKIAMFAUSER
aws_secret_access_key = mfaUserSecret
`
	require.NoError(t, os.WriteFile(cfgFile, []byte(cfgContent), 0o600))
	require.NoError(t, os.WriteFile(credFile, []byte(credContent), 0o600))
	t.Setenv("AWS_CONFIG_FILE", cfgFile)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ENDPOINT_URL_STS", stsURL)
	t.Cleanup(func() { SetAWSProfile("", "") })
}

func TestAWSProfile(t *testing.T) {
	sts := &fakeSTS{}
	srv := httptest.NewServer(sts)
	defer srv.Close()
	setupAWSTestConfig(t, srv.URL)
	ctx := context.Background()

	t.Run("Profile without MFA", func(t *testing.T) {
		SetAWSProfile("plain", "")
		cfg, err := loadAWSConfig(ctx)
		require.NoError(t, err)
		assert.Equal(t, "eu-central-1", cfg.Region)
		creds, err := cfg.Credentials.Retrieve(ctx)
		require.NoError(t, err)
		assert.Equal(t, "AKIAPLAIN", creds.AccessKeyID)
		assert.Nil(t, sts.lastRequest(), "STS should not be called")
	})
	t.Run("Profile with role and MFA", func(t *testing.T) {
		SetAWSProfile("mfarole", testMFAToken)
		cfg, err := loadAWSConfig(ctx)
		require.NoError(t, err)
		creds, err := cfg.Credentials.Retrieve(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ASIAROLE", creds.AccessKeyID)
		req := sts.lastRequest()
		require.NotNil(t, req)
		assert.Equal(t, "AssumeRole", req["Action"])
		assert.Equal(t, testMFAToken, req["TokenCode"])
		assert.Equal(t, testMFASerial, req["SerialNumber"])
	})
	t.Run("Profile with MFA without role", func(t *testing.T) {
		SetAWSProfile("mfauser", testMFAToken)
		cfg, err := loadAWSConfig(ctx)
		require.NoError(t, err)
		creds, err := cfg.Credentials.Retrieve(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ASIASESSION", creds.AccessKeyID)
		assert.Equal(t, "stsSession", creds.SessionToken)
		req := sts.lastRequest()
		require.NotNil(t, req)
		assert.Equal(t, "GetSessionToken", req["Action"])
		assert.Equal(t, testMFAToken, req["TokenCode"])
		assert.Equal(t, testMFASerial, req["SerialNumber"])
	})
	t.Run("Config is cached", func(t *testing.T) {
		SetAWSProfile("mfauser", testMFAToken)
		cfg1, err := loadAWSConfig(ctx)
		require.NoError(t, err)
		_, err = cfg1.Credentials.Retrieve(ctx)
		require.NoError(t, err)
		calls := len(sts.requests)
		cfg2, err := loadAWSConfig(ctx)
		require.NoError(t, err)
		_, err = cfg2.Credentials.Retrieve(ctx)
		require.NoError(t, err)
		assert.Len(t, sts.requests, calls, "MFA token must not be used twice")
	})
	t.Run("Unknown profile", func(t *testing.T) {
		SetAWSProfile("doesnotexist", "")
		_, err := loadAWSConfig(ctx)
		require.Error(t, err)
		_, err = ConnectToKMS()
		require.Error(t, err)
		_, err = ConnectToSecretsManager()
		require.Error(t, err)
		_, err = GetRDSAuthToken(testRDSHost, "", testRDSUser)
		require.Error(t, err)
	})
	t.Run("Connect KMS and Secrets Manager with profile", func(t *testing.T) {
		SetAWSProfile("plain", "")
		kmsSvc, err := ConnectToKMS()
		require.NoError(t, err)
		assert.NotNil(t, kmsSvc)
		smSvc, err := ConnectToSecretsManager()
		require.NoError(t, err)
		assert.NotNil(t, smSvc)
	})
}
