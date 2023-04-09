package dblib

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tommi2day/gomodules/ldaplib"
	"github.com/tommi2day/gomodules/test"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ldapOrganisation = "TNS Ltd"
const LdapDomain = "oracle.local"
const LdapBaseDn = "dc=oracle,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigPassword = "config"

const ldaptns = ` 
XE.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
XE1.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1))
)
XE2.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE2.local))
)
`

const ldaptns2 = ` 
XE2.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE2))
)
XE.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
`

const ldapora = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (oid:1389:1636, ldap:389)
DIRECTORY_SERVER_TYPE = OID
`

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var results []*ldap.Entry
	var sslport int
	var fileTnsEntries TNSEntries
	var ldapTnsEntries TNSEntries
	var lc *ldaplib.LdapConfigType

	test.Testinit(t)
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	ldapAdmin := test.TestData
	//nolint gosec
	err = os.WriteFile(ldapAdmin+"/ldap.ora", []byte(ldapora), 0644)
	require.NoErrorf(t, err, "Create test ldap.ora failed")

	t.Run("Parse ldap.ora", func(t *testing.T) {
		oraclecontext, ldapservers := ReadLdapOra(ldapAdmin)
		e := 2
		l := len(ldapservers)
		assert.NotEmpty(t, oraclecontext, "Conext should not be empty")
		assert.Equal(t, e, l, "ldapservers should have exact %d entries", 2)
		if l == e {
			s := ldapservers[0]
			expected := "oid:1389:1636"
			actual := fmt.Sprintf("%s:%d:%d", s.Hostname, s.Port, s.SSLPort)
			assert.Equal(t, expected, actual, "ldap entry 1 not match")
			s = ldapservers[1]
			expected = "ldap:389:0"
			actual = fmt.Sprintf("%s:%d:%d", s.Hostname, s.Port, s.SSLPort)
			assert.Equal(t, expected, actual, "ldap entry 2 not match")
		}
	})

	// prepare or skip container based tests
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer destroyLdapContainer(ldapContainer)

	base := LdapBaseDn
	server, sslport = getLdapHostAndPort(ldapContainer, "636/tcp")
	lc = ldaplib.NewConfig(server, sslport, true, true, base, 20*time.Second)
	context := ""

	t.Run("Ldap Connect", func(t *testing.T) {
		t.Logf("Connect '%s' using SSL on port %d", LdapAdminUser, sslport)
		err = lc.Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, lc.Conn, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, lc.Conn, "returned object ist not ldap connection")
		if lc.Conn == nil {
			t.Fatalf("No valid Connection, terminate")
			return
		}
	})

	t.Run("Get Oracle Context", func(t *testing.T) {
		context, err = GetOracleContext(lc, base)
		expected := "cn=OracleContext," + LdapBaseDn
		assert.NotEmptyf(t, context, "Oracle Context not found")
		assert.Equal(t, expected, context, "OracleContext not as expected")
		if context == "" {
			t.Fatalf("Oracle Context object not found, terminate")
			return
		}
		t.Logf("Oracle Context: %s", context)
	})
	domain := ""
	t.Run("Write TNS Entries", func(t *testing.T) {
		err = os.Chdir(test.TestDir)
		require.NoErrorf(t, err, "ChDir failed")

		// create test file to load
		tnsAdmin = TESTDATA
		filename := tnsAdmin + "/ldap_file.ora"
		//nolint gosec
		err = os.WriteFile(filename, []byte(ldaptns), 0644)
		require.NoErrorf(t, err, "Create test ldap_file.ora failed")
		t.Logf("load from %s", filename)

		// read entries from file
		fileTnsEntries, domain, err = GetTnsnames(filename, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
		if err != nil {
			t.Fatalf("tns load returned error: %s ", err)
			return
		}

		// write entries to ldap
		var workstatus TWorkStatus
		workstatus, err = WriteLdapTns(lc, fileTnsEntries, domain, context)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
		expected := len(fileTnsEntries)
		actual := workstatus[sNew]
		require.Equal(t, expected, actual, "Not all Records has been added")
		t.Logf("%d Entries added", actual)
	})

	if err != nil {
		t.Fatalf("need Write TNS to proceed")
		return
	}

	t.Run("Ldap TNS Search", func(t *testing.T) {
		results, err = lc.Search(context, "(objectclass=orclNetService)", []string{"DN"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		actual := len(results)
		assert.Greaterf(t, actual, 0, "Zero Entries")
		t.Logf("Returned %d entries", actual)
	})
	t.Run("Ldap TNS Read", func(t *testing.T) {
		ldapTnsEntries, err = ReadLdapTns(lc, context)
		require.NoErrorf(t, err, "Ldap Read returned error:%v", err)
		actual := len(ldapTnsEntries)
		expected := len(fileTnsEntries)
		assert.Equal(t, expected, actual, "Entry Count differs")
	})
	t.Run("Modify TNS Entries", func(t *testing.T) {
		err = os.Chdir(test.TestDir)
		require.NoErrorf(t, err, "ChDir failed")

		// create test file to load
		tnsAdmin = test.TestData
		filename := tnsAdmin + "/ldap_file2.ora"
		//nolint gosec
		err = os.WriteFile(filename, []byte(ldaptns2), 0644)
		require.NoErrorf(t, err, "Create test ldap_file2.ora failed")
		t.Logf("load from %s", filename)

		// read entries from file
		fileTnsEntries, domain, err = GetTnsnames(filename, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
		if err != nil {
			t.Fatalf("tns load returned error: %s ", err)
			return
		}
		require.Equal(t, 2, len(fileTnsEntries), "update TNS should have 2 entries")
		// write entries to ldap
		var workstatus TWorkStatus
		workstatus, err = WriteLdapTns(lc, fileTnsEntries, domain, context)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
		o := workstatus[sOK]
		n := workstatus[sNew]
		m := workstatus[sMod]
		d := workstatus[sDel]
		s := workstatus[sSkip]
		assert.Equal(t, 1, o, "One OK expected")
		assert.Equal(t, 0, n, "No Adds expected")
		assert.Equal(t, 1, m, "One mod expected")
		assert.Equal(t, 1, d, "One del expected")
		assert.Equal(t, 0, s, "One skip expected")
	})
}
