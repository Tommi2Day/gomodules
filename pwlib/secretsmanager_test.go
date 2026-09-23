package pwlib

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secretsManagerTest1 = "test"
const secretsManagerTest2 = "app/service"
const secretsManagerTestKMS = "app/kms"
const testPasswordField = "password"

// secretsManagerKeyID returns the KMS key ID of a secret
func secretsManagerKeyID(t *testing.T, svc *secretsmanager.Client, secretID string) string {
	out, err := svc.DescribeSecret(context.TODO(), &secretsmanager.DescribeSecretInput{SecretId: aws.String(secretID)})
	require.NoError(t, err)
	return aws.ToString(out.KmsKeyId)
}

func TestSecretsManager(t *testing.T) {
	var svc *secretsmanager.Client
	if os.Getenv("SKIP_SECRETSMANAGER") != "" {
		t.Skip("Skipping Secrets Manager testing in CI environment")
	}
	smContainer, err := prepareSecretsManagerContainer()
	defer common.DestroyDockerContainer(smContainer)
	require.NoErrorf(t, err, "Secrets Manager Server not available")
	require.NotNil(t, smContainer, "Prepare failed")
	if err != nil || smContainer == nil {
		t.Fatal("Secrets Manager Server not available")
	}
	host, port := common.GetContainerHostAndPort(smContainer, "5000/tcp")
	address := fmt.Sprintf("http://%s:%d", host, port)

	_ = os.Setenv("AWS_ACCESS_KEY_ID", "abcdef")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "abcdefSecret")
	_ = os.Setenv("AWS_DEFAULT_REGION", "eu-central-1")
	_ = os.Setenv("SECRETSMANAGER_ENDPOINT", address)
	defer func() {
		_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
		_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		_ = os.Unsetenv("AWS_DEFAULT_REGION")
		_ = os.Unsetenv("SECRETSMANAGER_ENDPOINT")
	}()

	t.Run("SecretsManager Connect", func(t *testing.T) {
		svc, err = ConnectToSecretsManager()
		require.NoErrorf(t, err, "Connect to Secrets Manager failed")
		require.NotNilf(t, svc, "Connect to Secrets Manager failed")
	})
	if svc == nil {
		t.Fatal("Connect to Secrets Manager failed")
	}

	t.Run("SecretsManager Write", func(t *testing.T) {
		data := map[string]interface{}{
			testPasswordField: "Awssm123",
		}
		err = SecretsManagerWrite(svc, secretsManagerTest1, data)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("SecretsManager Write nested path", func(t *testing.T) {
		data := map[string]interface{}{
			testPasswordField: "Awssm456",
		}
		err = SecretsManagerWrite(svc, secretsManagerTest2, data)
		require.NoErrorf(t, err, "Write returned error: %v", err)
	})
	t.Run("SecretsManager Read", func(t *testing.T) {
		value, rerr := SecretsManagerRead(svc, secretsManagerTest1)
		require.NoErrorf(t, rerr, "Read returned error: %v", rerr)
		assert.Containsf(t, value, "Awssm123", "unexpected secret value %q retrieved", value)
	})
	t.Run("SecretsManager Read wrong secret", func(t *testing.T) {
		value, rerr := SecretsManagerRead(svc, "does-not-exist")
		require.Error(t, rerr, "Read should return an error")
		require.Emptyf(t, value, "Secret value should be empty")
	})
	t.Run("SecretsManager ReadJSON", func(t *testing.T) {
		data, rerr := SecretsManagerReadJSON(svc, secretsManagerTest1)
		require.NoErrorf(t, rerr, "ReadJSON returned error: %v", rerr)
		require.NotNilf(t, data, "Secrets Manager data is nil")
		value, success := data["password"].(string)
		require.True(t, success, "Key password not found")
		assert.Equalf(t, "Awssm123", value, "unexpected password value %q retrieved from Secrets Manager", value)
	})
	t.Run("SecretsManager Write update existing", func(t *testing.T) {
		data := map[string]interface{}{
			testPasswordField: "Awssm789",
		}
		err = SecretsManagerWrite(svc, secretsManagerTest1, data)
		require.NoErrorf(t, err, "Write update returned error: %v", err)
		var data2 map[string]interface{}
		data2, err = SecretsManagerReadJSON(svc, secretsManagerTest1)
		require.NoErrorf(t, err, "ReadJSON returned error: %v", err)
		value, success := data2["password"].(string)
		require.True(t, success, "Key password not found")
		assert.Equalf(t, "Awssm789", value, "unexpected password value %q retrieved from Secrets Manager", value)
	})
	t.Run("SecretsManager Write with custom KMS key", func(t *testing.T) {
		// moto serves all services on one port, so the KMS key can be created there as well
		t.Setenv("KMS_ENDPOINT", address)
		defer func() { KmsEndpoint = "" }()
		kmsSvc, kerr := ConnectToKMS()
		require.NoError(t, kerr)
		key1, kerr := GenKMSKey(kmsSvc, "", "SecretsManagerKey1", nil)
		require.NoError(t, kerr)
		key2, kerr := GenKMSKey(kmsSvc, "", "SecretsManagerKey2", nil)
		require.NoError(t, kerr)
		defer func() { SecretsManagerKMSKeyID = "" }()
		data := map[string]interface{}{testPasswordField: "AwssmKms1"}

		// new secret gets key1
		SecretsManagerKMSKeyID = *key1.KeyMetadata.KeyId
		err = SecretsManagerWrite(svc, secretsManagerTestKMS, data)
		require.NoErrorf(t, err, "Write with KMS key returned error: %v", err)
		assert.Equal(t, *key1.KeyMetadata.KeyId, secretsManagerKeyID(t, svc, secretsManagerTestKMS))

		// existing secret is switched to key2
		SecretsManagerKMSKeyID = *key2.KeyMetadata.KeyId
		data[testPasswordField] = "AwssmKms2"
		err = SecretsManagerWrite(svc, secretsManagerTestKMS, data)
		require.NoErrorf(t, err, "Write with changed KMS key returned error: %v", err)
		assert.Equal(t, *key2.KeyMetadata.KeyId, secretsManagerKeyID(t, svc, secretsManagerTestKMS))
		value, rerr := SecretsManagerRead(svc, secretsManagerTestKMS)
		require.NoError(t, rerr)
		assert.Contains(t, value, "AwssmKms2")
	})
	t.Run("SecretsManager List", func(t *testing.T) {
		var entries []string
		entries, err = SecretsManagerList(svc)
		t.Logf("Secrets Manager List returned entries: %v", entries)
		require.NoErrorf(t, err, "List returned error: %v", err)
		assert.Containsf(t, entries, secretsManagerTest1, "Expected secret %s not found in %v", secretsManagerTest1, entries)
		assert.Containsf(t, entries, secretsManagerTest2, "Expected secret %s not found in %v", secretsManagerTest2, entries)
	})
	t.Run("SecretsManager GetPassword", func(t *testing.T) {
		app := "test_get_pass_secretsmanager"
		pc := NewConfig(app, test.TestData, test.TestData, app, typeAWSSM)
		pass, gerr := pc.GetPassword(secretsManagerTest1, "password")
		expected := "Awssm789"
		assert.NoErrorf(t, gerr, "Got unexpected error: %s", gerr)
		assert.Equal(t, expected, pass, "Answer not expected. exp:%s,act:%s", expected, pass)
	})
	t.Run("SecretsManager GetPassword fail", func(t *testing.T) {
		app := "test_get_pass_secretsmanager_fail"
		pc := NewConfig(app, test.TestData, test.TestData, app, typeAWSSM)
		pass, gerr := pc.GetPassword("does-not-exist", "password")
		assert.Error(t, gerr, "Should have failed")
		assert.Emptyf(t, pass, "pass Should be empty, but have %s", pass)
		if gerr != nil {
			t.Log(gerr)
		}
	})
}
