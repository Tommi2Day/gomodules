package pwlib

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	log "github.com/sirupsen/logrus"
)

// AWSProfile is the optional AWS shared config profile used for KMS and Secrets Manager.
// If empty, the AWS SDK default credential chain (incl. AWS_PROFILE) is used.
var AWSProfile = ""

// AWSMFATokenProvider optionally returns the current MFA token code, if the profile requires MFA.
// It is used for profiles with role_arn and mfa_serial (AssumeRole) as well as for
// profiles with mfa_serial only (GetSessionToken for IAM users).
var AWSMFATokenProvider func() (string, error)

var (
	awsConfigMu    sync.Mutex
	awsConfigCache *aws.Config
)

// SetAWSProfile sets the AWS profile and an optional MFA token for KMS and Secrets Manager.
// An empty profile falls back to the AWS SDK default credential chain,
// an empty mfaToken disables MFA token handling.
func SetAWSProfile(profile string, mfaToken string) {
	var provider func() (string, error)
	if mfaToken != "" {
		provider = func() (string, error) { return mfaToken, nil }
	}
	SetAWSProfileWithTokenProvider(profile, provider)
}

// SetAWSProfileWithTokenProvider sets the AWS profile and an optional MFA token provider,
// e.g. stscreds.StdinTokenProvider to prompt for the token on demand
func SetAWSProfileWithTokenProvider(profile string, tokenProvider func() (string, error)) {
	awsConfigMu.Lock()
	defer awsConfigMu.Unlock()
	AWSProfile = profile
	AWSMFATokenProvider = tokenProvider
	awsConfigCache = nil
	log.Debugf("use AWS profile '%s', MFA token provider set: %t", profile, tokenProvider != nil)
}

// ResetAWSConfig drops the cached AWS configuration, so the next connect loads it again
func ResetAWSConfig() {
	awsConfigMu.Lock()
	defer awsConfigMu.Unlock()
	awsConfigCache = nil
}

// loadAWSConfig loads the AWS configuration for the configured profile and MFA token.
// With MFA the configuration is cached, because an MFA token can only be used once and
// KMS and Secrets Manager clients should share the resulting credentials.
func loadAWSConfig(ctx context.Context) (aws.Config, error) {
	awsConfigMu.Lock()
	defer awsConfigMu.Unlock()
	if awsConfigCache != nil {
		return *awsConfigCache, nil
	}

	var opts []func(*config.LoadOptions) error
	if AWSProfile != "" {
		log.Debugf("load AWS config with profile %s", AWSProfile)
		opts = append(opts, config.WithSharedConfigProfile(AWSProfile))
	}
	tokenProvider := AWSMFATokenProvider
	if tokenProvider != nil {
		opts = append(opts, config.WithAssumeRoleCredentialOptions(func(o *stscreds.AssumeRoleOptions) {
			o.TokenProvider = tokenProvider
		}))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("cannot load AWS config: %w", err)
	}

	if tokenProvider != nil {
		// IAM user with MFA and without role: AssumeRole isn't used, so get a session token instead
		if sc, ok := sharedConfigFromSources(cfg.ConfigSources); ok && sc.RoleARN == "" && sc.MFASerial != "" {
			log.Debugf("profile %s requires MFA without role, use STS GetSessionToken", sc.Profile)
			cfg.Credentials = aws.NewCredentialsCache(&mfaSessionProvider{
				client:        sts.NewFromConfig(cfg),
				serialNumber:  sc.MFASerial,
				tokenProvider: tokenProvider,
			})
		}
	}
	if tokenProvider != nil {
		awsConfigCache = &cfg
	}
	return cfg, nil
}

// sharedConfigFromSources returns the shared config profile resolved by config.LoadDefaultConfig
func sharedConfigFromSources(sources []interface{}) (config.SharedConfig, bool) {
	for _, src := range sources {
		if sc, ok := src.(config.SharedConfig); ok {
			return sc, true
		}
	}
	return config.SharedConfig{}, false
}

// mfaSessionProvider retrieves temporary credentials with STS GetSessionToken using an MFA token
type mfaSessionProvider struct {
	client        *sts.Client
	serialNumber  string
	tokenProvider func() (string, error)
}

// Retrieve implements aws.CredentialsProvider
func (p *mfaSessionProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	token, err := p.tokenProvider()
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("cannot get MFA token: %w", err)
	}
	out, err := p.client.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		SerialNumber: aws.String(p.serialNumber),
		TokenCode:    aws.String(token),
	})
	if err != nil {
		return aws.Credentials{}, checkOperationError(err)
	}
	if out == nil || out.Credentials == nil {
		return aws.Credentials{}, errors.New("GetSessionToken returned no credentials")
	}
	c := out.Credentials
	creds := aws.Credentials{
		AccessKeyID:     aws.ToString(c.AccessKeyId),
		SecretAccessKey: aws.ToString(c.SecretAccessKey),
		SessionToken:    aws.ToString(c.SessionToken),
		Source:          "pwlibMFASessionProvider",
	}
	if c.Expiration != nil {
		creds.CanExpire = true
		creds.Expires = *c.Expiration
	}
	return creds, nil
}
