package dblib

import (
	"github.com/sijms/go-ora/v2/network"
	log "github.com/sirupsen/logrus"
)

// HaveOerr checks if there is an oracle error
func HaveOerr(err error) (isOerr bool, code int, msg string) {
	isOerr = false
	code = 0
	msg = ""
	if oerr, ok := err.(*network.OracleError); ok {
		code = oerr.ErrCode
		msg = oerr.ErrMsg
		log.Debugf("is Oracle Error Code: %d, Msg: %s\n", code, msg)
		isOerr = true
	}
	return
}
