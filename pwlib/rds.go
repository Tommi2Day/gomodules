package pwlib

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/tommi2day/gomodules/common"

	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	log "github.com/sirupsen/logrus"
)

// RDSDefaultPort is used if the RDS endpoint contains no port
var RDSDefaultPort = "5432"

// RDSRegion is the optional AWS region for RDS auth tokens, default is the region of the AWS config
var RDSRegion = ""

// GetRDSAuthToken returns an IAM authentication token for the given RDS endpoint (host[:port]) and database user.
// The token is valid for 15 minutes and is used as password for the database login.
// Credentials are taken from the AWS config, see SetAWSProfile.
func GetRDSAuthToken(endpoint string, region string, dbUser string) (token string, err error) {
	if endpoint == "" || dbUser == "" {
		return "", errors.New("endpoint or dbUser is empty")
	}
	if _, _, serr := net.SplitHostPort(endpoint); serr != nil {
		endpoint = net.JoinHostPort(endpoint, RDSDefaultPort)
	}
	ctx := context.TODO()
	cfg, err := loadAWSConfig(ctx)
	if err != nil {
		log.Warnf("cannot load AWS config for RDS auth token: %v", err)
		return "", err
	}
	if region == "" {
		region = common.GetStringEnv("RDS_REGION", RDSRegion)
	}
	if region == "" {
		region = cfg.Region
	}
	if region == "" {
		err = errors.New("no AWS region configured for RDS auth token")
		log.Warn(err)
		return "", err
	}
	log.Debugf("build RDS auth token for %s@%s in region %s", dbUser, endpoint, region)
	token, err = auth.BuildAuthToken(ctx, endpoint, region, dbUser, cfg.Credentials)
	if err != nil {
		err = fmt.Errorf("cannot build RDS auth token: %w", err)
		log.Warn(err)
		return "", err
	}
	return token, nil
}
