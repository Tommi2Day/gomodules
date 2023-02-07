package pwlib

import (
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	defaultRsaKeySize = 2048
	defaultMethod     = "go"
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
}

var label = []byte("")

// PwConfig Encryption configuration
var PwConfig PassConfig

// SetConfig set encryption configuration
func SetConfig(appname string, datadir string, keydir string, keypass string) {
	log.Debug("SetConfig entered")
	log.Debugf("A:%s, P:%s, D:%s, K:%s", appname, keypass, datadir, keydir)
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
	cryptedfile := datadir + "/" + appname + ".pw"
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
	PwConfig.Method = defaultMethod
	PwConfig.KeySize = defaultRsaKeySize
}
