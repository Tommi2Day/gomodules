package dblib

import (
	"os"
	"strings"
	"testing"

	"github.com/tommi2day/gomodules/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/test"
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
)
XE.SID =(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SID = XESID))
)
XE.error = Error
`

const sqlnetcontent = `
NAMES.DEFAULT_DOMAIN=local
NAMES.DIRECTORY_PATH=(TNSNAMES,EZCONNECT,LDAP)
WALLET_LOCATION =
    (SOURCE =
      (METHOD = FILE)
      (METHOD_DATA =
        (DIRECTORY = "/etc/oracle/wallet")
      )
    )
SSL_VERSION = 1.2
SSL_SERVER_DN_MATCH = Yes
SSL_CLIENT_AUTHENTICATION = True
SSL_CIPHER_SUITES= (SSL_RSA_WITH_RC4_128_SHA)
`
const entryCount = 6

var tnsAdmin = "testdata"

func TestParseTns(t *testing.T) {
	var err error
	test.InitTestDirs()
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	err = common.WriteStringToFile(tnsAdmin+"/sqlnet.ora", sqlnetcontent)
	require.NoErrorf(t, err, "Create test sqlnet.ora failed")
	err = common.WriteStringToFile(tnsAdmin+"/tnsnames.ora", tnsnamesora)
	require.NoErrorf(t, err, "Create test tnsnames.ora failed")
	err = common.WriteStringToFile(tnsAdmin+"/ifile.ora", ifileora)
	require.NoErrorf(t, err, "Create test ifile.ora failed")
	t.Run("Check TNS_ADMIN", func(t *testing.T) {
		var actual string
		actual, err = CheckTNSadmin(tnsAdmin)
		// replace windows path sep
		actual = strings.ReplaceAll(actual, "\\", "/")
		assert.NoErrorf(t, err, "unexpected Error %s", err)
		assert.Equal(t, tnsAdmin, actual, "Value not the same")
	})
	t.Run("Parse SQLNet.Ora", func(t *testing.T) {
		namesDomain, namesPath, sslInfo := ReadSQLNetOra(tnsAdmin)
		assert.NotEmpty(t, namesDomain, "Names domain should not empty")
		expected := 3
		actual := len(namesPath)
		assert.Equal(t, expected, actual, "NamesPath should have %d entries", expected)
		assert.Equal(t, "1.2", sslInfo.Version, "SSL_VERSION should be 1.2")
		assert.Equal(t, "/etc/oracle/wallet", sslInfo.WalletLocation, "WalletLocation should be /etc/oracle/wallet")
		assert.Equal(t, "SSL_RSA_WITH_RC4_128_SHA", sslInfo.Ciphers, "SSL_CIPHER_SUITES should be SSL_RSA_WITH_RC4_128_SHA")
		assert.True(t, sslInfo.ClientAthentication, "SSL_CLIENT_AUTHENTICATION should be true")
		assert.True(t, sslInfo.ServerDNMatch, "SSL_SERVER_DN_MATCH should be true")
	})
	domain, _, sslInfo := ReadSQLNetOra(tnsAdmin)
	t.Logf("Default Domain: '%s'", domain)
	filename := tnsAdmin + "/tnsnames.ora"
	t.Logf("load from %s", filename)
	walletDir := sslInfo.WalletLocation
	t.Logf("Wallet_Location: %s", walletDir)
	tnsEntries, domain, err := GetTnsnames(filename, true)
	t.Run("Parse TNSNames.ora", func(t *testing.T) {
		require.Error(t, err, "Parsing should have an error")
	})
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
				name:    "XE-SID",
				alias:   "XE.SID",
				success: true,
				service: "XESID",
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
				alias:   "XE.xx.xx",
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
		location := e.Location
		expectedLocation := "ifile.ora Line: 6"
		expectedDesc := `(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)`
		assert.Equal(t, strings.TrimSpace(expectedDesc), strings.TrimSpace(actualDesc), "Description not expected")
		assert.Equal(t, expectedLocation, location, "Location not expected")
		t.Logf("Location: %s", e.Location)
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
	const jdbcprefix = "jdbc:oracle:thin:@"
	const jdbcaddr1 = "(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))" +
		"(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=XE)))"
	t.Run("Test JDBC Output normal", func(t *testing.T) {
		desc := `(DESCRIPTION =
	(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE)))`
		actual, err := GetJDBCUrl(desc)
		expected := jdbcprefix + jdbcaddr1
		assert.NoError(t, err, "GetJDBCUrl failed")
		assert.NotEmptyf(t, actual, "JDBC Url empty")
		assert.Equal(t, expected, actual, "JDBC Url not expected")
	})
	const jdbcdesc = "(DESCRIPTION=(CONNECT_TIMEOUT=15)(RETRY_COUNT=20)(RETRY_DELAY=3)"
	const jdbc3 = "(TRANSPORT_CONNECT_TIMEOUT=3)"
	const jdbc3a = "(TRANSPORT_CONNECT_TIMEOUT=3000)"
	const jdbcaddr2 = "(ADDRESS_LIST=(LOAD_BALANCE=ON)(FAILOVER=ON)" +
		"(ADDRESS=(PROTOCOL=TCP)(HOST=vdb1.ora.local)(PORT=1672))" +
		"(ADDRESS=(PROTOCOL=TCP)(HOST=vdb2.ora.local)(PORT=1672)))" +
		"(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=Test.local)))"
	t.Run("Test JDBC Output with TRANSPORT_CONNECT_TIMEOUT", func(t *testing.T) {
		desc := `
(DESCRIPTION =
	(CONNECT_TIMEOUT=15)
	(RETRY_COUNT=20)
	(RETRY_DELAY=3)
	(transport_connect_timeout=3)
	(ADDRESS_LIST =
		(LOAD_BALANCE=ON)
		(FAILOVER=ON)
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb1.ora.local)(PORT=1672))
		(ADDRESS=(PROTOCOL=TCP)(HOST=vdb2.ora.local)(PORT=1672))
	)
	(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = Test.local))
)
`
		actual, err := GetJDBCUrl(desc)
		expected := jdbcprefix + jdbcdesc + jdbc3a + jdbcaddr2
		assert.NoError(t, err, "GetJDBCUrl failed")
		assert.NotEmptyf(t, actual, "JDBC Url empty")
		assert.Equal(t, expected, actual, "JDBC Url not expected")
	})
	t.Run("Test JDBC Output already replaced", func(t *testing.T) {
		desc := jdbcdesc + jdbc3a + jdbcaddr2
		actual, err := GetJDBCUrl(desc)
		expected := jdbcprefix + jdbcdesc + jdbc3a + jdbcaddr2
		assert.NoError(t, err, "GetJDBCUrl failed")
		assert.NotEmptyf(t, actual, "JDBC Url empty")
		assert.Equal(t, expected, actual, "JDBC Url not expected")
	})
	t.Run("Test JDBC Output NoReplace", func(t *testing.T) {
		desc := jdbcdesc + jdbc3 + jdbcaddr2
		ModifyJDBCTransportConnectTimeout = false
		actual, err := GetJDBCUrl(desc)
		expected := jdbcprefix + jdbcdesc + jdbc3 + jdbcaddr2
		assert.NoError(t, err, "GetJDBCUrl failed")
		assert.NotEmptyf(t, actual, "JDBC Url empty")
		assert.Equal(t, expected, actual, "JDBC Url not expected")
	})
}
