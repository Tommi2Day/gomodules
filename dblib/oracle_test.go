package dblib

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	ora "github.com/sijms/go-ora/v2"
	"github.com/sijms/go-ora/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
)

const DBUSER = "system"
const DBPASSWORD = "XE-manager21"
const TIMEOUT = 5
const port = "21521"
const repo = "docker.io/gvenzl/oracle-xe"
const repoTag = "21.3.0-slim"
const containerName = "dblib-oracle"
const containerTimeout = 240
const TESTDATA = "testdata"

var dbhost = common.GetEnv("DB_HOST", "127.0.0.1")
var connectora = fmt.Sprintf("XE.local=(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XEPDB1)))", dbhost, port)
var target string

// makeOerr create a pseudo ORA Errorcode
func makeOerr(code int, msg string) *network.OracleError {
	e := &network.OracleError{
		ErrCode: code,
		ErrMsg:  msg,
	}
	return e
}

// prepareContainer create an Oracle Docker Container
func prepareContainer() (pool *dockertest.Pool, dbContainer *dockertest.Resource, err error) {
	dbContainer = nil
	pool = nil
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	pool, err = dockertest.NewPool("")
	if err != nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}
	vendorImagePrefix := common.GetEnv("VENDOR_IMAGE_PREFIX", "")
	repoString := vendorImagePrefix + repo

	fmt.Printf("Try to start docker container for %s:%s\n", repoString, repoTag)
	dbContainer, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        repoTag,
		Env: []string{
			"ORACLE_PASSWORD=" + DBPASSWORD,
		},
		Name:         containerName,
		ExposedPorts: []string{"1521"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"1521": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})

	if err != nil {
		err = fmt.Errorf("error starting oracle docker container: %v", err)
		return
	}

	pool.MaxWait = containerTimeout * time.Second
	// hostAndPort = dbContainer.GetHostPort(port + "/tcp")
	target = fmt.Sprintf("oracle://%s:%s@%s:%s/xepdb1", "system", DBPASSWORD, dbhost, port)
	fmt.Printf("Wait to successfully connect to db with %s (max %ds)...\n", target, containerTimeout)
	start := time.Now()
	if err = pool.Retry(func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("oracle", target)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		fmt.Printf("Could not connect to DB Container: %s", err)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("DB Container is available after %s\n", elapsed.Round(time.Millisecond))
	err = nil
	return
}
func destroyContainer(pool *dockertest.Pool, dbContainer *dockertest.Resource) {
	if err := pool.Purge(dbContainer); err != nil {
		fmt.Printf("Could not purge resource: %s\n", err)
	}
}

func TestWithOracle(t *testing.T) {
	if os.Getenv("SKIP_ORACLE") != "" {
		t.Skip("Skipping ORACLE testing in CI environment")
	}
	const alias = "XE.local"

	tnsAdmin := TESTDATA
	filename := tnsAdmin + "/connect.ora"
	//_ = os.Chdir(tnsAdmin)
	//nolint gosec
	_ = os.WriteFile(filename, []byte(connectora), 0644)

	t.Logf("load from %s", filename)
	domain := GetDefaultDomain(tnsAdmin)
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

	pool, dbContainer, err := prepareContainer()
	require.NotNil(t, dbContainer, "Prepare failed")
	defer destroyContainer(pool, dbContainer)

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
		var db *sql.DB
		connect := target
		t.Logf("connect with %s\n", connect)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.NoErrorf(t, err, "Connect failed: %v", err)
		assert.IsType(t, &sql.DB{}, db, "Returned wrong type")
		result, err := SelectOneStringValue(db, "select to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual")
		assert.NoErrorf(t, err, "Select returned error::%v", err)
		assert.NotEmpty(t, result)
		t.Logf("Sysdate: %s", result)
	})
	t.Run("Check tns connect", func(t *testing.T) {
		var db *sql.DB
		connect := ora.BuildJDBC(DBUSER, DBPASSWORD, desc, urlOptions)
		t.Logf("connect with %s\n", connect)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.NoErrorf(t, err, "Connect failed: %s", err)
		assert.IsType(t, &sql.DB{}, db, "Returned wrong type")
	})
	t.Run("Check dummy connect", func(t *testing.T) {
		var db *sql.DB
		connect := ora.BuildJDBC("dummy", "dummy", desc, urlOptions)
		t.Logf("connect with dummy user to %s\n", desc)
		db, err = DBConnect("oracle", connect, TIMEOUT)
		assert.ErrorContainsf(t, err, "ORA-01017", "returned unexpected error: %v", err)
		assert.IsType(t, &sql.DB{}, db, "Returned wrong type")
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
