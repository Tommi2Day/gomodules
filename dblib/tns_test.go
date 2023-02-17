package dblib

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
	"os"
	"strings"
	"testing"
)

const tnsnamesora = `
# Test ifile relative
ifile=ifile.ora
DB_T.local=
  (DESCRIPTION=
    (CONNECT_TIMEOUT=15)
    (TRANSPORT_CONNECT_TIMEOUT=3)
    (ADDRESS_LIST=
      (FAILOVER=on)
      (LOAD_BALANCE=on)
      (ADDRESS=
        (PROTOCOL=TCP)
        (HOST=tdb1.ora.local)
        (PORT=1562)
      )
      (ADDRESS=
        (PROTOCOL=TCP)
        (HOST=tdb2.ora.local)
        (PORT=1562)
      )
    )
    (CONNECT_DATA=
      (SERVER=dedicated)
      (SERVICE_NAME=DB_T.local)
    )
  )


DB_V.local =(DESCRIPTION =
	(CONNECT_TIMEOUT=15)
	(RETRY_COUNT=20)
	(RETRY_DELAY=3)
	(TRANSPORT_CONNECT_TIMEOUT=3)
	(ADDRESS_LIST =
		(LOAD_BALANCE=ON)
		(FAILOVER=ON)
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb1.ora.local)(PORT=1672))
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb2.ora.local)(PORT=1672))
	)
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = DB_V.local))
)
`

const ifileora = `
XE =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE-ohne))
)
XE.local =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
XE1 =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1))
)`
const sqlnetora = `
NAMES.DEFAULT_DOMAIN=local
NAMES.DIRECTORY_PATH=(TNSNAMES,EZCONNECT)
`
const entryCount = 5

func TestParseTns(t *testing.T) {
	var err error

	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	tnsAdmin := "testdata"
	//nolint gosec
	err = os.WriteFile(tnsAdmin+"/sqlnet.ora", []byte(sqlnetora), 0644)
	//nolint gosec
	err = os.WriteFile(tnsAdmin+"/sqlnet.ora", []byte(sqlnetora), 0644)
	require.NoErrorf(t, err, "Create test sqlnet.ora failed")
	//nolint gosec
	err = os.WriteFile(tnsAdmin+"/tnsnames.ora", []byte(tnsnamesora), 0644)
	require.NoErrorf(t, err, "Create test tnsnames.ora failed")
	//nolint gosec
	err = os.WriteFile(tnsAdmin+"/ifile.ora", []byte(ifileora), 0644)
	require.NoErrorf(t, err, "Create test ifile.ora failed")

	domain := GetDefaultDomain(tnsAdmin)
	t.Logf("Default Domain: '%s'", domain)
	filename := tnsAdmin + "/tnsnames.ora"
	t.Logf("load from %s", filename)
	tnsEntries, domain, err := GetTnsnames(filename, true)
	t.Run("Parse TNSNames.ora", func(t *testing.T) {
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
	})
	if err != nil {
		t.Logf("load returned error: %s ", err)
		return
	}
	t.Run("Count Entries", func(t *testing.T) {
		countEntries := len(tnsEntries)
		expected := entryCount
		actual := countEntries
		assert.Equal(t, expected, actual, "Count not expected")
	})
	t.Run("Check entry", func(t *testing.T) {
		type testTableType struct {
			name    string
			alias   string
			success bool
			service string
		}
		for _, testconfig := range []testTableType{
			{
				name:    "XE-full",
				alias:   "XE.local",
				success: true,
				service: "XE",
			},
			{
				name:    "XE-short",
				alias:   "XE",
				success: true,
				service: "XE",
			},
			{
				name:    "XE1-short-invalid",
				alias:   "XE1",
				success: false,
				service: "",
			},
			{
				name:    "XE+full-invalid",
				alias:   "XE1.local",
				success: false,
				service: "",
			},
			{
				name:    "XE+invalid domain",
				alias:   "XE" + ".xx.xx",
				success: false,
				service: "",
			},
			{
				name:    "novalue",
				alias:   "",
				success: false,
				service: "",
			},
		} {
			t.Run(testconfig.name, func(t *testing.T) {
				e, ok := GetEntry(testconfig.alias, tnsEntries, domain)
				if testconfig.success {
					assert.True(t, ok, "Alias %s not found", testconfig.alias)
					name := strings.ToUpper(e.Name)
					assert.True(t, strings.HasPrefix(name, strings.ToUpper(testconfig.alias)), "entry not related to given alias %s", testconfig.alias)
					assert.Equalf(t, testconfig.service, e.Service, "entry returned wrong service ('%s' <>'%s)", e.Service, testconfig.service)
				} else {
					assert.False(t, ok, "Alias %s found, but shouldnt be", testconfig.alias)
				}
			})
		}
	})

	alias := "XE"
	t.Run("Check entry value", func(t *testing.T) {
		e, ok := GetEntry(alias, tnsEntries, domain)
		assert.True(t, ok, "Alias %s not found", alias)
		actualDesc := e.Desc
		expectedDesc := `(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)`
		assert.Equal(t, strings.TrimSpace(expectedDesc), strings.TrimSpace(actualDesc), "Description not expected")
	})
	t.Run("Check Server Entry", func(t *testing.T) {
		e, found := tnsEntries[alias]
		assert.True(t, found, "Alias not found")
		actual := len(e.Servers)
		expected := 1
		assert.Equal(t, expected, actual, "Server Count not expected")
		if actual > 0 {
			server := e.Servers[0]
			assert.NotEmpty(t, server.Host, "Host ist empty")
			assert.NotEmpty(t, server.Port, "Port ist empty")
		}
	})
}
