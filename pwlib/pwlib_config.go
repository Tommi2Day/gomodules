package pwlib

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/go-openssl/v4"
)

const (
	defaultRsaKeySize = 2048
	typeGO            = "go"
	typeOpenssl       = "openssl"
	typePlain         = "plain"
	typeEnc           = "b64"
	typeVault         = "vault"
	typeGPG           = "gpg"
	typeGopass        = "gopass"
	typeKMS           = "kms"
	defaultMethod     = typeGO
	extGo             = "gp"
	extOpenssl        = "pw"
	extPlain          = "plain"
	extB64            = "b64"
	privPemExt        = ".pem"
	pubPemExt         = ".pub"
	extGPG            = "gpg"
	extKMS            = "kms"
	pubGPGExt         = ".pub.gpg"
	privGPGExt        = ".priv.gpg"
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
	KMSKeyID        string
	CaseSensitive   bool
}

var label = []byte("")
var pubExt = pubPemExt
var privExt = privPemExt

// SSLDigest specifies the digest algorithm used by OpenSSL for deriving encryption keys, set to SHA-256.
var SSLDigest = openssl.BytesToKeySHA256

// NewConfig set encryption configuration
func NewConfig(appname string, datadir string, keydir string, keypass string, method string) (passConfig *PassConfig) {
	var ext string
	config := PassConfig{}
	log.Debug("NewConfig entered")
	log.Debugf("A:%s, P:%s, D:%s, K:%s, M:%s", appname, keypass, datadir, keydir, method)
	// default names
	defaultDir := path.Dir(keydir)
	if defaultDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home, _ = os.Getwd()
		}
		defaultDir = path.Join(home, ".pwcli")
	}

	if method == "" {
		method = defaultMethod
	}
	switch method {
	case typeOpenssl:
		ext = extOpenssl
	case typeGO:
		ext = extGo
	case typePlain:
		ext = extPlain
	case typeEnc:
		ext = extB64
	case typeVault:
		ext = extPlain
	case typeKMS:
		ext = extKMS
	case typeGPG, typeGopass:
		ext = extGPG
		privExt = privGPGExt
		pubExt = pubGPGExt
		if keypass == "" {
			keypass = os.Getenv("GPG_PASSPHRASE")
		}
	default:
		log.Warnf("invalid method %s, use method %s", method, defaultMethod)
		method = defaultMethod
		ext = extGo
	}
	if datadir == "" {
		datadir = defaultDir
	}
	if keydir == "" {
		keydir = defaultDir
	}
	if keypass == "" {
		keypass = appname
	}

	cryptedfile := datadir + "/" + appname + "." + ext
	privatekeyfile := keydir + "/" + appname + privExt
	pubkeyfile := keydir + "/" + appname + pubExt
	plainfile := datadir + "/" + appname + ".plain"
	sessionpassfile := keydir + "/" + appname + ".dat"

	// set global configuration defaults, any part can be overwritten
	config.AppName = appname
	config.DataDir = datadir
	config.KeyDir = keydir
	config.KeyPass = keypass
	config.CryptedFile = cryptedfile
	config.PrivateKeyFile = privatekeyfile
	config.PubKeyFile = pubkeyfile
	config.PlainTextFile = plainfile
	config.SessionPassFile = sessionpassfile
	config.Method = method
	config.KeySize = defaultRsaKeySize
	config.SSLDigest = SSLDigest
	config.KMSKeyID = ""
	config.CaseSensitive = false
	return &config
}
