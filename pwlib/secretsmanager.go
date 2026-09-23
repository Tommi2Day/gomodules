package pwlib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"
)

// SecretsManagerEndpoint is the alternative endpoint for the AWS Secrets Manager service
var SecretsManagerEndpoint = ""

// SecretsManagerKMSKeyID is the optional customer managed KMS key (ID, ARN or alias) used by SecretsManagerWrite.
// If empty, Secrets Manager uses the default key aws/secretsmanager.
var SecretsManagerKMSKeyID = ""

// ConnectToSecretsManager establishes a connection to AWS Secrets Manager
func ConnectToSecretsManager() (svc *secretsmanager.Client, err error) {
	log.Debugf("Connect to Secrets Manager")
	cfg, err := loadAWSConfig(context.TODO())
	if err != nil {
		log.Warnf("cannot connect to Secrets Manager: %v", err)
		return nil, err
	}
	ep := common.GetStringEnv("SECRETSMANAGER_ENDPOINT", "")
	if ep != "" {
		SecretsManagerEndpoint = ep
	}
	svc = secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) {
		if SecretsManagerEndpoint != "" {
			log.Debugf("use Secrets Manager Endpoint %s", SecretsManagerEndpoint)
			o.BaseEndpoint = aws.String(SecretsManagerEndpoint)
		}
	})
	return svc, nil
}

// SecretsManagerRead reads the raw secret string of the given secret from AWS Secrets Manager
func SecretsManagerRead(svc *secretsmanager.Client, secretID string) (value string, err error) {
	if svc == nil {
		return "", errors.New("secrets manager service is nil")
	}
	if secretID == "" {
		return "", errors.New("secretID is empty")
	}
	log.Debugf("read secret %s from Secrets Manager", secretID)
	output, err := svc.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return "", checkOperationError(err)
	}
	if output == nil || output.SecretString == nil {
		return "", fmt.Errorf("no secret string returned for %s", secretID)
	}
	log.Debugf("got secret %s", secretID)
	return *output.SecretString, nil
}

// SecretsManagerReadJSON reads a secret from AWS Secrets Manager and decodes it as a JSON object
func SecretsManagerReadJSON(svc *secretsmanager.Client, secretID string) (data map[string]interface{}, err error) {
	value, err := SecretsManagerRead(svc, secretID)
	if err != nil {
		return nil, err
	}
	data = map[string]interface{}{}
	if err = json.Unmarshal([]byte(value), &data); err != nil {
		return nil, fmt.Errorf("secret %s is not a valid JSON object: %w", secretID, err)
	}
	log.Debugf("decoded JSON secret %s", secretID)
	return data, nil
}

// SecretsManagerWrite writes JSON encoded data to the given secret, creating it if it does not exist yet
func SecretsManagerWrite(svc *secretsmanager.Client, secretID string, data map[string]interface{}) (err error) {
	if svc == nil {
		return errors.New("secrets manager service is nil")
	}
	if secretID == "" {
		return errors.New("secretID is empty")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal secret data: %w", err)
	}
	keyID := common.GetStringEnv("SECRETSMANAGER_KMS_KEY_ID", SecretsManagerKMSKeyID)
	log.Debugf("write secret %s to Secrets Manager", secretID)
	if keyID == "" {
		_, err = svc.PutSecretValue(context.TODO(), &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(secretID),
			SecretString: aws.String(string(payload)),
		})
	} else {
		// UpdateSecret sets value and KMS key together, so existing secrets are switched to the given key
		log.Debugf("use KMS key %s for secret %s", keyID, secretID)
		_, err = svc.UpdateSecret(context.TODO(), &secretsmanager.UpdateSecretInput{
			SecretId:     aws.String(secretID),
			SecretString: aws.String(string(payload)),
			KmsKeyId:     aws.String(keyID),
		})
	}
	var notFound *types.ResourceNotFoundException
	if errors.As(err, &notFound) {
		log.Debugf("secret %s not found, creating it", secretID)
		input := &secretsmanager.CreateSecretInput{
			Name:         aws.String(secretID),
			SecretString: aws.String(string(payload)),
		}
		if keyID != "" {
			input.KmsKeyId = aws.String(keyID)
		}
		_, err = svc.CreateSecret(context.TODO(), input)
	}
	if err != nil {
		return checkOperationError(err)
	}
	log.Debugf("write secret %s successfully", secretID)
	return nil
}

// SecretsManagerList lists the names of all secrets available in AWS Secrets Manager
func SecretsManagerList(svc *secretsmanager.Client) (entries []string, err error) {
	if svc == nil {
		return nil, errors.New("secrets manager service is nil")
	}
	log.Debugf("list Secrets Manager secrets")
	paginator := secretsmanager.NewListSecretsPaginator(svc, &secretsmanager.ListSecretsInput{})
	for paginator.HasMorePages() {
		page, pErr := paginator.NextPage(context.TODO())
		if pErr != nil {
			return nil, checkOperationError(pErr)
		}
		for _, s := range page.SecretList {
			if s.Name != nil {
				entries = append(entries, *s.Name)
			}
		}
	}
	log.Debugf("Secrets Manager list returned %d entries", len(entries))
	return entries, nil
}

// GetAWSSMSecret reads a secret path as system and returns its keys and values as plaintext format
func GetAWSSMSecret(secretID string, endpoint string) (content string, err error) {
	log.Debugf("Secrets Manager Read entered for secret '%s'", secretID)
	if endpoint != "" {
		SecretsManagerEndpoint = endpoint
	}
	svc, err := ConnectToSecretsManager()
	if err != nil {
		return
	}
	data, err := SecretsManagerReadJSON(svc, secretID)
	if err != nil {
		return "", err
	}
	sysKey := strings.ReplaceAll(secretID, ":", "_")
	for k, v := range data {
		content += fmt.Sprintf("%s:%s:%v\n", sysKey, k, v)
	}
	log.Debug("Secrets Manager Read OK")
	return content, nil
}
