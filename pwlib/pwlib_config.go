package pwlib

import (
	"os"

	"github.com/Luzifer/go-openssl/v4"

	log "github.com/sirupsen/logrus"
)

const (
	defaultRsaKeySize = 2048
	typeGO            = "go"
	typeOpenssl       = "openssl"
	defaultMethod     = typeGO
	extGo             = "gp"
	extOpenssl        = "pw"
)

// PassConfig Type for encryption configuration
type PassConfig struct {
	AppName         string
	DataDir         string
	KeyDir          string
	KeyPass         string
	CryptedFile     string
	PrivateKeyFile  string
	PubKeyFile      string
	PlainTextFile   string
	SessionPassFile string
	Method          string
	KeySize         int
	SSLDigest       openssl.CredsGenerator
}

var label = []byte("")

// PwConfig Encryption configuration
var PwConfig PassConfig

// SSLDigest variable helds common digist algor
var SSLDigest = openssl.BytesToKeySHA256

// SetConfig set encryption configuration
func SetConfig(appname string, datadir string, keydir string, keypass string, method string) {
	var ext string
	log.Debug("SetConfig entered")
	log.Debugf("A:%s, P:%s, D:%s, K:%s, M:%s", appname, keypass, datadir, keydir, method)
	// default names
	wd, _ := os.Getwd()
	etc := wd + "/etc"
	if datadir == "" {
		datadir = etc
	}
	if keydir == "" {
		keydir = etc
	}
	if keypass == "" {
		keypass = appname
	}
	if method == "" {
		method = defaultMethod
	}
	switch method {
	case typeOpenssl:
		ext = extOpenssl
	case typeGO:
		ext = extGo
	default:
		log.Warnf("invalid method %s, use method %s", method, defaultMethod)
		method = defaultMethod
		ext = extGo
	}
	cryptedfile := datadir + "/" + appname + "." + ext
	privatekeyfile := keydir + "/" + appname + ".pem"
	pubkeyfile := keydir + "/" + appname + ".pub"
	plainfile := datadir + "/" + appname + ".plain"
	sessionpassfile := keydir + "/" + appname + ".dat"

	// set global configuration defaults, any part can be overwritten
	PwConfig.AppName = appname
	PwConfig.DataDir = datadir
	PwConfig.KeyDir = keydir
	PwConfig.KeyPass = keypass
	PwConfig.CryptedFile = cryptedfile
	PwConfig.PrivateKeyFile = privatekeyfile
	PwConfig.PubKeyFile = pubkeyfile
	PwConfig.PlainTextFile = plainfile
	PwConfig.SessionPassFile = sessionpassfile
	PwConfig.Method = method
	PwConfig.KeySize = defaultRsaKeySize
	PwConfig.SSLDigest = SSLDigest
}
