package pwlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	gopassEnvStoreDir   = "PASSWORD_STORE_DIR"
	gopassDefaultSubDir = ".local/share/gopass/stores/root" //nolint:gosec // path constant, not a credential
	gopassSecretExt     = ".gpg"
	gopassAgeExt        = ".age"

	// GopassCryptoGPG selects GPG encryption (default, secrets stored as .gpg files).
	GopassCryptoGPG = "gpg"
	// GopassCryptoAge selects age encryption (secrets stored as .age files).
	GopassCryptoAge = "age"
)

// gopassExtFor returns the file extension for the given crypto type.
func gopassExtFor(cryptoType string) string {
	if cryptoType == GopassCryptoAge {
		return gopassAgeExt
	}
	return gopassSecretExt
}

// GopassStoreDir resolves the gopass store directory.
// If storeDir is non-empty it is returned as-is.
// Otherwise PASSWORD_STORE_DIR env var is checked, then ~/.local/share/gopass/stores/root.
func GopassStoreDir(storeDir string) (string, error) {
	if storeDir != "" {
		return storeDir, nil
	}
	if dir := os.Getenv(gopassEnvStoreDir); dir != "" {
		log.Debugf("gopass store dir from env: %s", dir)
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, filepath.FromSlash(gopassDefaultSubDir))
	log.Debugf("gopass store dir default: %s", dir)
	return dir, nil
}

// GopassList returns all secret names in the store relative to storeDir, without the crypto extension.
// cryptoType selects which secrets to list: GopassCryptoGPG (.gpg), GopassCryptoAge (.age),
// or "" to list both. Secret names use forward slashes as path separators regardless of OS.
// Pass storeDir="" to use the default store.
func GopassList(storeDir, cryptoType string) (secrets []string, err error) {
	storeDir, err = GopassStoreDir(storeDir)
	if err != nil {
		return
	}
	err = filepath.Walk(storeDir, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		ext := matchGopassExt(p, cryptoType)
		if ext == "" {
			return nil
		}
		rel, relErr := filepath.Rel(storeDir, p)
		if relErr != nil {
			return relErr
		}
		secrets = append(secrets, filepath.ToSlash(strings.TrimSuffix(rel, ext)))
		return nil
	})
	if err != nil {
		err = fmt.Errorf("list gopass store %s failed: %w", storeDir, err)
		return
	}
	log.Debugf("listed %d secrets in gopass store %s", len(secrets), storeDir)
	return
}

// matchGopassExt returns the matching extension for a file path given the requested cryptoType,
// or "" if the file should be skipped.
func matchGopassExt(p, cryptoType string) string {
	switch cryptoType {
	case GopassCryptoAge:
		if strings.HasSuffix(p, gopassAgeExt) {
			return gopassAgeExt
		}
	case GopassCryptoGPG:
		if strings.HasSuffix(p, gopassSecretExt) {
			return gopassSecretExt
		}
	default: // "" = both
		if strings.HasSuffix(p, gopassAgeExt) {
			return gopassAgeExt
		}
		if strings.HasSuffix(p, gopassSecretExt) {
			return gopassSecretExt
		}
	}
	return ""
}

// GopassReadRaw decrypts a gopass secret and returns its full raw content.
// For GPG secrets keyFile is the armored private key file and keypass unlocks it.
// For age secrets keyFile is the age identity file and keypass is ignored.
// Pass storeDir="" to use the default store.
func GopassReadRaw(storeDir, secretName, keyFile, keypass, cryptoType string) (content string, err error) {
	storeDir, err = GopassStoreDir(storeDir)
	if err != nil {
		return
	}
	secretPath := filepath.Join(storeDir, filepath.FromSlash(secretName)+gopassExtFor(cryptoType))
	switch cryptoType {
	case GopassCryptoAge:
		content, err = AgeDecryptFileAuto(secretPath, keyFile, keypass)
	default: // GPG
		if keyFile == "" {
			content, err = GPGDecryptFileAuto(secretPath, keypass)
		} else {
			content, err = GPGDecryptFile(secretPath, keyFile, keypass, "")
		}
	}
	if err != nil {
		err = fmt.Errorf("read gopass secret %s failed: %w", secretName, err)
		return
	}
	log.Debugf("read raw gopass secret %s", secretName)
	return
}

// GopassRead decrypts a gopass secret and returns the first line (the password value).
// For GPG secrets keyFile is the armored private key file and keypass unlocks it.
// For age secrets keyFile is the age identity file and keypass is ignored.
// Pass storeDir="" to use the default store.
func GopassRead(storeDir, secretName, keyFile, keypass, cryptoType string) (secret string, err error) {
	var raw string
	raw, err = GopassReadRaw(storeDir, secretName, keyFile, keypass, cryptoType)
	if err != nil {
		return
	}
	// gopass format: first line is the secret value; remaining lines are YAML metadata
	lines := strings.SplitN(raw, "\n", 2)
	secret = strings.TrimRight(lines[0], "\r")
	log.Debugf("read gopass secret %s", secretName)
	return
}

// GopassWrite encrypts content and stores it as a gopass secret at secretName.
// content should have the password as the first line; additional lines may contain YAML metadata.
// For GPG secrets keyFile is the armored public key file.
// For age secrets keyFile is the age recipients file (public key).
// Parent directories are created automatically.
// Pass storeDir="" to use the default store.
func GopassWrite(storeDir, secretName, content, keyFile, cryptoType string) (err error) {
	storeDir, err = GopassStoreDir(storeDir)
	if err != nil {
		return
	}
	secretPath := filepath.Join(storeDir, filepath.FromSlash(secretName)+gopassExtFor(cryptoType))
	//nolint gosec
	if err = os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
		err = fmt.Errorf("create directory for gopass secret %s failed: %w", secretName, err)
		return
	}
	tmpFile, tmpErr := os.CreateTemp("", "gopass-*.tmp")
	if tmpErr != nil {
		err = fmt.Errorf("create temp file failed: %w", tmpErr)
		return
	}
	tmpName := tmpFile.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, tmpErr = tmpFile.WriteString(content); tmpErr != nil {
		_ = tmpFile.Close()
		err = fmt.Errorf("write temp file failed: %w", tmpErr)
		return
	}
	_ = tmpFile.Close()
	switch cryptoType {
	case GopassCryptoAge:
		err = AgeEncryptFile(tmpName, secretPath, keyFile)
	default: // GPG
		err = GPGEncryptFile(tmpName, secretPath, keyFile)
	}
	if err != nil {
		err = fmt.Errorf("write gopass secret %s failed: %w", secretName, err)
		return
	}
	log.Debugf("wrote gopass secret %s to %s", secretName, secretPath)
	return
}

// ---- config & auto-detection -----------------------------------------------

const (
	gopassEnvConfig    = "GOPASS_CONFIG"         //nolint:gosec // env-var name, not a credential
	gopassEnvXDGConfig = "XDG_CONFIG_HOME"       //nolint:gosec // env-var name, not a credential
	gopassConfigSubDir = ".config/gopass/config" //nolint:gosec // path constant, not a credential

	// gopassCryptoAge is the gopass internal backend name for age.
	gopassCryptoAge = "age"

	// Marker files written by gopass into the store root.
	gopassAgeMarker = ".age-recipients"
	gopassGPGMarker = ".gpg-id"
)

// GopassStoreConfig holds the configuration for a single gopass store or mount.
type GopassStoreConfig struct {
	Path   string `yaml:"path"`
	Crypto string `yaml:"crypto"`
}

// GopassConfig is a parsed subset of the gopass configuration file.
type GopassConfig struct {
	Root   GopassStoreConfig            `yaml:"root"`
	Mounts map[string]GopassStoreConfig `yaml:"mounts"`
}

// GopassConfigPath returns the path to the gopass configuration file.
// It checks GOPASS_CONFIG, then XDG_CONFIG_HOME/gopass/config,
// then falls back to ~/.config/gopass/config.
func GopassConfigPath() (string, error) {
	if p := os.Getenv(gopassEnvConfig); p != "" {
		return p, nil
	}
	if xdg := os.Getenv(gopassEnvXDGConfig); xdg != "" {
		return filepath.Join(xdg, "gopass", "config"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, filepath.FromSlash(gopassConfigSubDir)), nil
}

// GopassReadConfig reads and parses the gopass configuration file.
// Pass configPath="" to resolve the path automatically via GopassConfigPath.
func GopassReadConfig(configPath string) (cfg *GopassConfig, err error) {
	if configPath == "" {
		configPath, err = GopassConfigPath()
		if err != nil {
			return
		}
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		err = fmt.Errorf("read gopass config %s failed: %w", configPath, err)
		return
	}
	cfg = &GopassConfig{}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		err = fmt.Errorf("parse gopass config %s failed: %w", configPath, err)
		cfg = nil
	}
	log.Debugf("read gopass config from %s", configPath)
	return
}

// GopassDetectCrypto determines the encryption type used by a gopass store.
// Detection is attempted in this order:
//  1. Presence of .age-recipients marker file in storeDir → age
//  2. Presence of .gpg-id marker file in storeDir → gpg
//  3. File-extension scan of storeDir (.age / .gpg)
//  4. Lookup in the gopass config file by store path
//
// Pass storeDir="" to use the default store.
func GopassDetectCrypto(storeDir string) (cryptoType string, err error) {
	storeDir, err = GopassStoreDir(storeDir)
	if err != nil {
		return
	}

	// 1. age marker file
	if _, e := os.Stat(filepath.Join(storeDir, gopassAgeMarker)); e == nil {
		cryptoType = GopassCryptoAge
		log.Debugf("detected crypto=%s from %s in %s", cryptoType, gopassAgeMarker, storeDir)
		return
	}

	// 2. GPG marker file
	if _, e := os.Stat(filepath.Join(storeDir, gopassGPGMarker)); e == nil {
		cryptoType = GopassCryptoGPG
		log.Debugf("detected crypto=%s from %s in %s", cryptoType, gopassGPGMarker, storeDir)
		return
	}

	// 3. extension scan
	cryptoType = detectCryptoByExtension(storeDir)
	if cryptoType != "" {
		log.Debugf("detected crypto=%s from file extensions in %s", cryptoType, storeDir)
		return
	}

	// 4. gopass config file
	cryptoType = detectCryptoFromConfig(storeDir)
	if cryptoType != "" {
		log.Debugf("detected crypto=%s from gopass config for %s", cryptoType, storeDir)
		return
	}

	err = fmt.Errorf("cannot detect crypto type for gopass store %s", storeDir)
	return
}

// detectCryptoByExtension walks storeDir and returns the crypto type of the
// first secret file found (.age → age, .gpg → gpg).
func detectCryptoByExtension(storeDir string) (cryptoType string) {
	_ = filepath.Walk(storeDir, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || cryptoType != "" {
			return walkErr
		}
		switch {
		case strings.HasSuffix(p, gopassAgeExt):
			cryptoType = GopassCryptoAge
		case strings.HasSuffix(p, gopassSecretExt):
			cryptoType = GopassCryptoGPG
		}
		return nil
	})
	return
}

// detectCryptoFromConfig reads the gopass config and looks up storeDir in the
// root store and all mounts. Returns "" when the config is unavailable or the
// store is not listed.
func detectCryptoFromConfig(storeDir string) string {
	cfg, err := GopassReadConfig("")
	if err != nil || cfg == nil {
		return ""
	}
	if ct := storeCryptoFromConfig(cfg.Root, storeDir); ct != "" {
		return ct
	}
	for _, mount := range cfg.Mounts {
		if ct := storeCryptoFromConfig(mount, storeDir); ct != "" {
			return ct
		}
	}
	return ""
}

// storeCryptoFromConfig maps a GopassStoreConfig entry to a GopassCrypto*
// constant when its path matches storeDir. Returns "" on no match.
func storeCryptoFromConfig(sc GopassStoreConfig, storeDir string) string {
	if sc.Path == "" || filepath.Clean(sc.Path) != filepath.Clean(storeDir) {
		return ""
	}
	switch sc.Crypto {
	case gopassCryptoAge:
		return GopassCryptoAge
	default: // "gpgcli", "gpg", "" → GPG is the gopass default
		return GopassCryptoGPG
	}
}

// GopassReadSecretLines reads a gopass secret and returns its content formatted
// as "secretName:field:value" lines, compatible with GetPassword / DecryptFile.
//
// The password on the first line is always returned as field "password".
// Additional YAML key-value pairs after the "---" separator are returned as
// their own lines, e.g.:
//
//	databases/prod/postgres:password:s3cr3t
//	databases/prod/postgres:username:dbuser
//	databases/prod/postgres:host:db.example.com
//
// Pass storeDir="" to use the default store.
func GopassReadSecretLines(storeDir, secretName, keyFile, keypass, cryptoType string) (content string, err error) {
	raw, err := GopassReadRaw(storeDir, secretName, keyFile, keypass, cryptoType)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimRight(raw, "\r\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		err = fmt.Errorf("gopass secret %s is empty", secretName)
		return
	}

	var result []string
	password := strings.TrimRight(lines[0], "\r")
	result = append(result, fmt.Sprintf("%s:password:%s", secretName, password))

	// parse YAML front matter that follows the "---" separator
	inYAML := false
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, "\r")
		if line == "---" {
			inYAML = true
			continue
		}
		if !inYAML {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 && parts[0] != "" {
			result = append(result, fmt.Sprintf("%s:%s:%s", secretName, parts[0], parts[1]))
		}
	}

	content = strings.Join(result, "\n")
	log.Debugf("read %d field(s) from gopass secret %s", len(result), secretName)
	return
}

// GetGopassSecret reads a single gopass secret and returns it as a
// "system:account:password" line, compatible with GetPassword.
// The secret path is split at the last "/" into system and account components.
// For GPG secrets keyFile is the armored private key file and keypass unlocks it.
// For age secrets keyFile is the age identity file and keypass is ignored.
// Pass storeDir="" to use the default store.
func GetGopassSecret(storeDir, secretName, keyFile, keypass, cryptoType string) (content string, err error) {
	var secret string
	secret, err = GopassRead(storeDir, secretName, keyFile, keypass, cryptoType)
	if err != nil {
		return
	}
	lastSlash := strings.LastIndex(secretName, "/")
	var system, account string
	if lastSlash < 0 {
		system = secretName
		account = secretName
	} else {
		system = secretName[:lastSlash]
		account = secretName[lastSlash+1:]
	}
	content = fmt.Sprintf("%s:%s:%s\n", system, account, secret)
	log.Debugf("got gopass secret for path %s", secretName)
	return
}
