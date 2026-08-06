// Package dblib collection of db func
package dblib

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// reProtocolTCPS matches a TCPS protocol entry in a tns description
var reProtocolTCPS = regexp.MustCompile(`(?i)PROTOCOL\s*=\s*TCPS`)

// LoadSSLConfig reads sqlnet.ora from tnsDir (the directory holding the active
// tnsnames.ora, the same way sqlplus resolves it via TNS_ADMIN) and makes the
// wallet and SSL settings available to SSLConnectOptions and GetJDBCUrl via
// TNSSSLconfig. Callers that need a password for a non auto-login wallet
// should set TNSSSLconfig.WalletPassword themselves after calling this, since
// sqlnet.ora never stores one.
func LoadSSLConfig(tnsDir string) {
	_, _, sslInfo := ReadSQLNetOra(tnsDir)
	if sslInfo.WalletLocation != "" {
		sslInfo.WalletLocation = resolveWalletLocation(sslInfo.WalletLocation, tnsDir)
	}
	TNSSSLconfig = sslInfo
}

// resolveWalletLocation expands ${TNS_ADMIN}/$TNS_ADMIN references (common in
// real sqlnet.ora files) and resolves a relative wallet directory against the
// directory sqlnet.ora was read from.
func resolveWalletLocation(loc string, tnsDir string) string {
	loc = strings.ReplaceAll(loc, "${TNS_ADMIN}", tnsDir)
	loc = strings.ReplaceAll(loc, "$TNS_ADMIN", tnsDir)
	if !filepath.IsAbs(loc) {
		loc = filepath.Join(tnsDir, loc)
	}
	return loc
}

// SSLConnectOptions builds go-ora connection options for wallet/TCPS support
// out of the parsed sqlnet.ora settings (see LoadSSLConfig), the same way
// sqlplus would use WALLET_LOCATION and SSL_SERVER_DN_MATCH from sqlnet.ora
// for a TCPS connect.
func SSLConnectOptions(tnsDesc string) map[string]string {
	options := map[string]string{}
	if reProtocolTCPS.MatchString(tnsDesc) {
		// go-ora only loads the wallet into the TLS session when SSL is
		// explicitly enabled at connection-config build time; relying on the
		// PROTOCOL=TCPS address alone (negotiated later, per server) is too
		// late for that.
		options["SSL"] = "TRUE"
	}
	wallet := TNSSSLconfig.WalletLocation
	if wallet == "" {
		return options
	}
	if _, err := os.Stat(wallet); err != nil {
		log.Warnf("configured wallet directory '%s' not found: %v", wallet, err)
		return options
	}
	options["WALLET"] = wallet
	if TNSSSLconfig.WalletPassword != "" {
		options["WALLET PASSWORD"] = TNSSSLconfig.WalletPassword
	}
	if TNSSSLconfig.ServerDNMatch {
		options["SSL VERIFY"] = "TRUE"
	} else {
		options["SSL VERIFY"] = "FALSE"
	}
	return options
}
