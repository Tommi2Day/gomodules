package pwlib

import (
	"encoding/base64"

	"github.com/tommi2day/gomodules/common"

	log "github.com/sirupsen/logrus"
)

// EncodeFile encodes a file using base64
func EncodeFile(plainFile string, targetFile string) (err error) {
	plainData := ""
	log.Debugf("Encrypt %s with B64 to %s", plainFile, targetFile)
	//nolint gosec
	plainData, err = common.ReadFileToString(plainFile)
	if err != nil {
		log.Debugf("Cannot read plaintext file %s:%s", plainFile, err)
		return
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(plainData))
	err = common.WriteStringToFile(targetFile, b64)
	if err != nil {
		log.Debugf("Cannot write: %s", err.Error())
		return
	}
	return
}

// DecodeFile decodes a file using base64
func DecodeFile(cryptedfile string) (content []byte, err error) {
	data := ""
	log.Debugf("decrypt b64 %s", cryptedfile)
	data, err = common.ReadFileToString(cryptedfile)
	if err != nil {
		log.Debugf("Cannot Read file '%s': %s", cryptedfile, err)
		return
	}
	bindata, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Debugf("decode base64 for %s failed: %s", cryptedfile, err)
		return
	}
	content = bindata
	return
}
