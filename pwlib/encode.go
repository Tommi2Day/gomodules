package pwlib

import (
	"encoding/base64"
	"os"

	log "github.com/sirupsen/logrus"
)

// EncodeFile encodes a file using base64
func EncodeFile(plainFile string, targetFile string) (err error) {
	var plaindata []byte
	log.Debugf("Encrypt %s with B64 to %s", plainFile, targetFile)
	//nolint gosec
	plaindata, err = os.ReadFile(plainFile)
	if err != nil {
		log.Debugf("Cannot read plaintext file %s:%s", plainFile, err)
		return
	}
	b64 := base64.StdEncoding.EncodeToString(plaindata)
	//nolint gosec
	err = os.WriteFile(targetFile, []byte(b64), 0644)
	if err != nil {
		log.Debugf("Cannot write: %s", err.Error())
		return
	}
	return
}

// DecodeFile decodes a file using base64
func DecodeFile(cryptedfile string) (content []byte, err error) {
	var data []byte
	log.Debugf("decrypt b64 %s", cryptedfile)
	//nolint gosec
	data, err = os.ReadFile(cryptedfile)
	if err != nil {
		log.Debugf("Cannot Read file '%s': %s", cryptedfile, err)
		return
	}
	bindata, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		log.Debugf("decode base64 for %s failed: %s", cryptedfile, err)
		return
	}
	content = bindata
	return
}
