package dblib

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/tommi2day/gomodules/common"

	"github.com/ory/dockertest/v3"
	"github.com/tommi2day/gomodules/test"

	ora "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const DBPDB = "FREEPDB1"
const DBUSER = "system"
const DBPASSWORD = "XE-manager21"
const TIMEOUT = 5

var DBhost = common.GetEnv("DB_HOST", "127.0.0.1")
var oracleContainer *dockertest.Resource
var connectora = fmt.Sprintf("%s.local=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=%s)))", DBPDB, DBhost, DBPort, DBPDB)
var target string

// makeOerr create a pseudo ORA Errorcode
func makeOerr(code int, msg string) *network.OracleError {
	e := &network.OracleError{
		ErrCode: code,
		ErrMsg:  msg,
	}
	return e
}

func TestWithOracle(t *testing.T) {
	const alias = DBPDB + ".local"

	test.InitTestDirs()
	tnsAdmin = test.TestData
	filename := tnsAdmin + "/connect.ora"
	//_ = os.Chdir(tnsAdmin)
	//nolint gosec
	_ = os.WriteFile(filename, []byte(connectora), 0644)

	t.Logf("load from %s", filename)
	domain, _ := ReadSqlnetOra(tnsAdmin)
	t.Logf("Default Domain: '%s'", domain)

	tnsEntries, d2, err := GetTnsnames(filename, true)
	t.Run("Parse TNSNames.ora", func(t *testing.T) {
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
	})
	if err != nil {
		t.Logf("load returned error: %s ", err)
		return
	}

	assert.Equalf(t, domain, d2, "Domain name mismatch '%s' -> '%s'", domain, d2)
	e, found := GetEntry(alias, tnsEntries, domain)
	require.True(t, found, "Alias not found")
	desc := common.RemoveSpace(e.Desc)

	if os.Getenv("SKIP_ORACLE") != "" {
		t.Skip("Skipping ORACLE testing in CI environment")
	}

	oracleContainer, err = prepareContainer()
	require.NoErrorf(t, err, "Oracle Server not available:%v", err)
	require.NotNil(t, oracleContainer, "Prepare failed")
	defer common.DestroyDockerContainer(oracleContainer)

	t.Run("Direct connect", func(t *testing.T) {
		var db *sql.DB
		t.Logf("connect to %s\n", target)
		db, err = sql.Open("oracle", target)
		assert.NoErrorf(t, err, "Open failed: %s", err)
		assert.IsType(t, &sql.DB{}, db, "Returned wrong type")
		err = db.Ping()
		assert.NoErrorf(t, err, "Connect failed: %s", err)
	})
	t.Run("connect with function", func(t *testing.T) {
		var db *sqlx.DB
		connect := target
		t.Logf("connect with %s\n", connect)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.NoErrorf(t, err, "Connect failed: %v", err)
		assert.IsType(t, &sqlx.DB{}, db, "Returned wrong type")
		result, err := SelectOneStringValue(db, "select to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual")
		assert.NoErrorf(t, err, "Select returned error::%v", err)
		assert.NotEmpty(t, result)
		t.Logf("Sysdate: %s", result)
	})
	t.Run("Check tns connect", func(t *testing.T) {
		var db *sqlx.DB
		connect := ora.BuildJDBC(DBUSER, DBPASSWORD, desc, urlOptions)
		t.Logf("connect with %s\n", connect)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.NoErrorf(t, err, "Connect failed: %s", err)
		assert.IsType(t, &sqlx.DB{}, db, "Returned wrong type")
	})
	t.Run("Check dummy connect", func(t *testing.T) {
		var db *sqlx.DB
		connect := ora.BuildJDBC("dummy", "dummy", desc, urlOptions)
		t.Logf("connect with dummy user to %s\n", desc)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.ErrorContainsf(t, err, "ORA-01017", "returned unexpected error: %v", err)
		assert.IsType(t, &sqlx.DB{}, db, "Returned wrong type")
	})
}

// TestGetVersion Test Version output should return a nonempty value
func TestHaveOerr(t *testing.T) {
	var oerr *network.OracleError
	var testErr = errors.New("test error")
	t.Run("Oracle Error", func(t *testing.T) {
		expectedCode := 1017
		expectedMsg := "ORA-01017: Invalid User or Password"
		oerr = makeOerr(expectedCode, expectedMsg)
		isOerr, actualCode, actualMsg := HaveOerr(oerr)
		assert.True(t, isOerr, "Oerr not detected")
		assert.Equal(t, expectedCode, actualCode, "Code doesnt match")
		assert.Equal(t, expectedMsg, actualMsg, "Msg doesnt match")
	})
	t.Run("Non Oracle Error", func(t *testing.T) {
		expectedCode := 0
		expectedMsg := ""
		isOerr, actualCode, actualMsg := HaveOerr(testErr)
		assert.False(t, isOerr, "Oerr false detected")
		assert.Equal(t, expectedCode, actualCode, "Code doesnt match")
		assert.Equal(t, expectedMsg, actualMsg, "Empty Msg expected")
	})
}
