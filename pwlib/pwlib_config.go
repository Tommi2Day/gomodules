package pwlib

import (
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/tommi2day/gomodules/common"

	"github.com/Luzifer/go-openssl/v4"
)

// PCmethods is a list of supported encryption methods
var PCmethods = []string{
	"go",
	"openssl",
	"plain",
	"b64",
	"vault",
	"gpg",
	"gopass",
	"kms",
	"age",
}

const (
	defaultRsaKeySize = 2048
	KeyTypeRSA        = "rsa"
	KeyTypeECDSA      = "ecdsa"
	KeyTypeGPG        = "gpg"
	KeyTypeAGE        = "age"
	KeyTypeUnknown    = "unknown"
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
	pubGPGExt         = ".pub.gpg"
	privGPGExt        = ".priv.gpg"
	pubAgeExt         = ".pub.age"
	privAgeExt        = ".priv.age"
	typeAge           = "age"
	extAge            = "age"
	// ... other types ...
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
	KeyType         string
}

var (
	label   = []byte("")
	pubExt  = pubPemExt
	privExt = privPemExt
	ext     = extOpenssl
)

// SSLDigest specifies the digest algorithm used by OpenSSL for deriving encryption keys, set to SHA-256.
var SSLDigest = openssl.BytesToKeySHA256

// NewConfig set encryption configuration
// NewConfig erstellt eine neue Verschlüsselungskonfiguration
func NewConfig(appname, datadir, keydir, keypass, method string) *PassConfig {
	log.Debug("NewConfig entered")
	log.Debugf("A:%s, P:%s, D:%s, K:%s, M:%s", appname, keypass, datadir, keydir, method)

	method = getValidMethod(method)
	defaultDir := getDefaultDir(keydir)

	ext, privExt, pubExt = getExtensionsForMethod(method)
	keypass = getKeypass(keypass, appname, method)

	datadir = getOrDefault(datadir, defaultDir)
	keydir = getOrDefault(keydir, defaultDir)

	config := &PassConfig{
		AppName:         appname,
		DataDir:         datadir,
		KeyDir:          keydir,
		KeyPass:         keypass,
		CryptedFile:     path.Join(datadir, appname+"."+ext),
		PrivateKeyFile:  path.Join(keydir, appname+privExt),
		PubKeyFile:      path.Join(keydir, appname+pubExt),
		PlainTextFile:   path.Join(datadir, appname+".plain"),
		SessionPassFile: path.Join(keydir, appname+".dat"),
		Method:          method,
		KeySize:         defaultRsaKeySize,
		SSLDigest:       SSLDigest,
		KMSKeyID:        "",
		CaseSensitive:   false,
		KeyType:         KeyTypeRSA,
	}

	return config
}

func getDefaultDir(keydir string) string {
	defaultDir := path.Dir(keydir)
	if defaultDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home, _ = os.Getwd()
		}
		defaultDir = path.Join(home, ".pwcli")
	}
	return defaultDir
}

func getValidMethod(method string) string {
	if ok, _ := common.InArray(method, PCmethods); ok {
		return method
	}
	log.Warnf("invalid method %s, use method %s", method, defaultMethod)
	return defaultMethod
}

func getExtensionsForMethod(method string) (ext, privExt, pubExt string) {
	switch method {
	case typeOpenssl:
		return extOpenssl, privPemExt, pubPemExt
	case typeGO:
		return extGo, privPemExt, pubPemExt
	case typePlain:
		return extPlain, privPemExt, pubPemExt
	case typeEnc:
		return extB64, privPemExt, pubPemExt
	case typeAge:
		return extAge, privAgeExt, pubAgeExt
	case typeGPG:
		return extGPG, privGPGExt, pubGPGExt
	case typeGopass, typeKMS, typeVault:
		return "", "", ""
	default:
		log.Warnf("invalid method %s, use method %s", method, defaultMethod)
		return extGo, privPemExt, pubPemExt
	}
}

func getKeypass(keypass, appname, method string) string {
	if keypass != "" {
		return keypass
	}
	pass := ""
	switch method {
	case typeGPG:
		pass = os.Getenv("GPG_PASSPHRASE")
		return pass
	case typeAge:
		pass = os.Getenv("AGE_PASSPHRASE")
		return pass
	}
	return appname
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
