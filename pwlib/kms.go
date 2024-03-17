package pwlib

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tommi2day/gomodules/common"

	"github.com/Luzifer/go-openssl/v4"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	log "github.com/sirupsen/logrus"
)

// KmsEndpoint is the alternative endpoint for the KMS service
var KmsEndpoint = ""

const aliasPrefix = "alias/"

// ConnectToKMS Establish a connection to AWS KMS
func ConnectToKMS() (svc *kms.Client) {
	log.Debugf("Connect to KMS")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	ep := common.GetStringEnv("KMS_ENDPOINT", "")
	if ep != "" {
		KmsEndpoint = ep
	}
	svc = kms.NewFromConfig(cfg, func(o *kms.Options) {
		if KmsEndpoint != "" {
			log.Debugf("use KMS Endpoint %s", KmsEndpoint)
			o.BaseEndpoint = aws.String(KmsEndpoint)
		}
	})
	return svc
}

func checkOperationError(err error) error {
	var oe *smithy.OperationError
	e := err
	if errors.As(err, &oe) {
		e = fmt.Errorf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
	}
	log.Debugf("OperationError:%v", e)
	return e
}

// ListKMSKeys List all KMS keys
func ListKMSKeys(svc *kms.Client) ([]types.KeyListEntry, error) {
	log.Debugf("List KMS Keys")
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	output, err := svc.ListKeys(context.TODO(), &kms.ListKeysInput{})
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	if output == nil {
		return nil, errors.New("listKeys returned nil")
	}
	keys := output.Keys
	log.Debugf("ListKeys returned %d entries", len(keys))
	return keys, nil
}

// GetKMSKeyIDs Get KMS IDs from key meta data
func GetKMSKeyIDs(metadata *types.KeyMetadata) (keyID string, keyARN string) {
	if metadata == nil {
		return
	}
	keyID = *metadata.KeyId
	keyARN = *metadata.Arn
	return
}

// DescribeKMSKey Describe a KMS key
func DescribeKMSKey(svc *kms.Client, keyID string) (*kms.DescribeKeyOutput, error) {
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	if keyID == "" {
		return nil, errors.New("keyID is empty")
	}
	log.Debugf("Describe KMS Key:%s", keyID)
	output, err := svc.DescribeKey(context.TODO(), &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	log.Debugf("Describe Key was OK")
	return output, nil
}

// ListKMSAliases List all KMS aliases
func ListKMSAliases(svc *kms.Client, keyID string) ([]types.AliasListEntry, error) {
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	ip := &kms.ListAliasesInput{}
	if keyID != "" {
		ip = &kms.ListAliasesInput{
			KeyId: aws.String(keyID),
		}
	}
	log.Debugf("List KMS Key Aliases for %s", keyID)
	output, err := svc.ListAliases(context.TODO(), ip)
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	if output == nil {
		return nil, errors.New("listAliases returned nil")
	}
	aliases := output.Aliases
	log.Debugf("ListAliases returned %d entries", len(aliases))
	return aliases, nil
}

// CreateKMSAlias Create a KMS alias
func CreateKMSAlias(svc *kms.Client, aliasName string, targetKeyID string) (*kms.CreateAliasOutput, error) {
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	if targetKeyID == "" {
		return nil, errors.New("targetKeyID is empty")
	}
	if aliasName == "" {
		return nil, errors.New("aliasName is empty")
	}
	log.Debugf("Create KMS Alias:%s for Key:%s", aliasName, targetKeyID)
	if !strings.HasPrefix(aliasName, aliasPrefix) {
		aliasName = aliasPrefix + aliasName
	}
	output, err := svc.CreateAlias(context.TODO(), &kms.CreateAliasInput{
		AliasName:   aws.String(aliasName),
		TargetKeyId: aws.String(targetKeyID),
	})
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	log.Debugf("Create Alias was OK")
	return output, nil
}

// DeleteKMSAlias Delete a KMS alias
func DeleteKMSAlias(svc *kms.Client, aliasName string) (*kms.DeleteAliasOutput, error) {
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	if aliasName == "" {
		return nil, errors.New("aliasName is empty")
	}
	if !strings.HasPrefix(aliasName, aliasPrefix) {
		aliasName = aliasPrefix + aliasName
	}
	log.Debugf("Delete KMS Alias:%s", aliasName)
	output, err := svc.DeleteAlias(context.TODO(), &kms.DeleteAliasInput{
		AliasName: aws.String(aliasName),
	})
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	log.Debugf("Delete Alias was OK")
	return output, nil
}

// GetKMSAliasIDs Get KMS AliasIDs from Alias Entry
func GetKMSAliasIDs(entry *types.AliasListEntry) (targetKeyID string, aliasName string, aliasARN string) {
	log.Debugf("Get KMS IDs from Aliias entry")
	if entry == nil {
		return
	}
	aliasName = *entry.AliasName
	targetKeyID = *entry.TargetKeyId
	aliasARN = *entry.AliasArn
	log.Debugf("Entry %s point to target %s", aliasName, targetKeyID)
	return
}

// DescribeKMSAlias Search and Describe a KMS alias
func DescribeKMSAlias(svc *kms.Client, aliasName string) (*types.AliasListEntry, error) {
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	if aliasName == "" {
		return nil, errors.New("aliasName is empty")
	}
	if !strings.HasPrefix(aliasName, "alias/") {
		aliasName = aliasPrefix + aliasName
	}
	log.Debugf("Describe KMS Alias:%s", aliasName)
	output, err := svc.ListAliases(context.TODO(), &kms.ListAliasesInput{})
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	if output == nil {
		return nil, errors.New("listAliases returned nil")
	}
	aliases := output.Aliases
	for _, a := range aliases {
		if *a.AliasName == aliasName {
			log.Debugf("Alias %s found", aliasName)
			return &a, nil
		}
	}
	log.Debugf("Alias %s not found", aliasName)
	return nil, fmt.Errorf("alias %s not found", aliasName)
}

// GenKMSKey Create a new KMS key
func GenKMSKey(svc *kms.Client, keyspec string, description string, tags map[string]string) (*kms.CreateKeyOutput, error) {
	log.Debugf("Create KMS Key")
	if svc == nil {
		return nil, errors.New("KMS service is nil")
	}
	var keytags []types.Tag
	for k, v := range tags {
		kt := types.Tag{
			TagKey:   aws.String(k),
			TagValue: aws.String(v),
		}
		keytags = append(keytags, kt)
	}
	keyOutput, err := svc.CreateKey(
		context.TODO(),
		&kms.CreateKeyInput{
			Description: aws.String(description),
			KeySpec:     types.KeySpec(keyspec),
			Tags:        keytags,
		},
	)
	if err != nil {
		e := checkOperationError(err)
		return nil, e
	}
	log.Debugf("Create Key was OK")
	return keyOutput, nil
}

// KMSEncryptString Encrypt a string using the KMS key
func KMSEncryptString(svc *kms.Client, keyID string, plaintext string) (string, error) {
	if svc == nil {
		return "", errors.New("KMS service is nil")
	}
	if keyID == "" {
		return "", errors.New("keyID is empty")
	}
	log.Debugf("Encrypt with KMS key:%s", keyID)
	output, err := svc.Encrypt(
		context.TODO(),
		&kms.EncryptInput{
			KeyId:     aws.String(keyID),
			Plaintext: []byte(plaintext),
		},
	)
	if err != nil {
		e := checkOperationError(err)
		return "", e
	}
	log.Debugf("Encrypt was OK")
	return string(output.CiphertextBlob), nil
}

// KMSDecryptString Decrypt a string using the KMS key
func KMSDecryptString(svc *kms.Client, keyID string, ciphertext string) (string, error) {
	if svc == nil {
		return "", errors.New("KMS service is nil")
	}
	if keyID == "" {
		return "", errors.New("keyID is empty")
	}
	if ciphertext == "" {
		return "", errors.New("ciphertext is empty")
	}
	log.Debugf("Decrypt with KMS key:%s", keyID)
	output, err := svc.Decrypt(
		context.TODO(),
		&kms.DecryptInput{
			KeyId:          aws.String(keyID),
			CiphertextBlob: []byte(ciphertext),
		},
	)
	if err != nil {
		e := checkOperationError(err)
		return "", e
	}
	log.Debugf("Decrypt was OK")
	return string(output.Plaintext), nil
}

// KMSEncryptFile Encrypt a file using the KMS key
func KMSEncryptFile(plainFile string, targetFile string, keyID string, sessionPassFile string) (err error) {
	const rb = 16
	log.Debugf("Encrypt %s with KMS key %s in OpenSSL compatible format", plainFile, keyID)
	if keyID == "" || plainFile == "" || targetFile == "" {
		err = fmt.Errorf("keyID, plainFile or targetFile is empty")
		log.Debug(err)
		return
	}
	svc := ConnectToKMS()
	if svc == nil {
		err = fmt.Errorf("cannot connect to KMS")
		log.Debug(err)
		return
	}
	random := make([]byte, rb)
	_, err = rand.Read(random)
	if err != nil {
		log.Debugf("Cannot generate session key:%s", err)
		return
	}
	sessionKey := base64.StdEncoding.EncodeToString(random)
	crypted, err := KMSEncryptString(svc, keyID, sessionKey)
	if err != nil {
		log.Errorf("Encrypting Keyfile failed: %v", err)
	}

	if len(sessionPassFile) > 0 {
		//nolint gosec
		err = os.WriteFile(sessionPassFile, []byte(crypted), 0644)
		if err != nil {
			log.Errorf("Cannot write session Key file %s:%v", sessionPassFile, err)
		}
	}

	//nolint gosec
	plaindata, err := os.ReadFile(plainFile)
	if err != nil {
		log.Debugf("Cannot read plaintext file %s:%s", plainFile, err)
		return
	}

	o := openssl.New()
	// openssl enc -e -aes-256-cbc -md sha246 -base64 -in $SOURCE -out $TARGET -pass pass:$PASSPHRASE
	encrypted, err := o.EncryptBytes(sessionKey, plaindata, SSLDigest)
	if err != nil {
		log.Errorf("cannot encrypt plaintext file %s:%s", plainFile, err)
		return
	}
	// write crypted output file
	//nolint gosec
	err = os.WriteFile(targetFile, encrypted, 0644)
	if err != nil {
		log.Errorf("Cannot write: %s", err.Error())
		return
	}
	return
}

// KMSDecryptFile Decrypt a file using the KMS key
func KMSDecryptFile(cryptedFile string, keyID string, sessionPassFile string) (content string, err error) {
	if cryptedFile == "" || sessionPassFile == "" || keyID == "" {
		err = fmt.Errorf("keyID, crypted filename or sessionpassfilename is empty")
		log.Debug(err)
		return
	}
	log.Debugf("decrypt %s with KMS key %s", cryptedFile, keyID)
	svc := ConnectToKMS()
	if svc == nil {
		err = fmt.Errorf("cannot connect to KMS")
		log.Debug(err)
		return
	}
	//nolint gosec
	cryptedData, err := os.ReadFile(cryptedFile)
	if err != nil {
		log.Debugf("cannot Read file '%s': %s", cryptedFile, err)
		return
	}
	encSessionKey := ""

	var sp []byte
	//nolint gosec
	sp, err = os.ReadFile(sessionPassFile)
	if err != nil {
		log.Debugf("cannot Read file '%s': %s", sessionPassFile, err)
		return
	}
	encSessionKey = string(sp)

	if err != nil {
		log.Debugf("Cannot Read file '%s': %s", sessionPassFile, err)
		return
	}

	sessionKey, err := KMSDecryptString(svc, keyID, encSessionKey)
	if err != nil {
		log.Debugf("decode session key failed:%s", err)
		return
	}
	// sk := string(sessionKey)
	log.Debug("Session key decrypted")

	// OPENSSL enc -d -aes-256-cbc -md sha256 -base64 -in $SOURCE -pass pass:$SESSIONKEY
	o := openssl.New()
	decoded, err := o.DecryptBytes(sessionKey, cryptedData, SSLDigest)
	if err != nil {
		log.Debugf("Cannot decrypt data from '%s': %s", cryptedFile, err)
		return
	}
	content = string(decoded)
	log.Debug("Decoding successfully")
	return
}
