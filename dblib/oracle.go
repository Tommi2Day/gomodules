package dblib

import (
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
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

// CheckWithOracle try a connect to oracle with dummy creds to get an ORA error.
// If this happens, the connect is working
func CheckWithOracle(dbuser string, dbpass string, tnsDesc string, timeout int) (ok bool, elapsed time.Duration, err error) {
	urlOptions := map[string]string{
		// "CONNECTION TIMEOUT": "3",
	}
	const defaultUser = "dummy"
	const defaultPassword = "dummy"
	ok = false
	if dbuser == "" {
		dbuser = defaultUser
	}
	if dbpass == "" {
		dbpass = defaultPassword
	}
	// jdbc url needs spaces stripped
	tnsDesc = strings.Join(strings.Fields(tnsDesc), "")
	url := go_ora.BuildJDBC(dbuser, dbpass, tnsDesc, urlOptions)
	log.Debugf("Try to connect %s@%s", dbuser, tnsDesc)
	start := time.Now()
	db, err := DBConnect("oracle", url, timeout)
	elapsed = time.Since(start)

	// check results
	if err != nil {
		// check error code, we expect 1017
		isOerr, code, _ := HaveOerr(err)
		if isOerr && code == 1017 && dbuser == defaultUser && dbpass == defaultPassword {
			ok = true
			log.Debugf("Got expected error code 1017, Connect OK")
		}
	} else {
		log.Debugf("Connection OK, test if db is open using select")
		sql := "select to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual"
		val, err := SelectOneStringValue(db, sql)
		log.Debugf("Query returned sysdate = %s", val)
		if err == nil {
			ok = true
		}
	}
	return
}
