package dblib

import (
	"github.com/tommi2day/gomodules/test"
	"os"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/ldaplib"
)

const ldapOrganisation = "TNS Ltd"
const LdapDomain = "oracle.local"
const LdapBaseDn = "dc=oracle,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigPassword = "config"

const ldaptns = ` 
XE =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE-ohne))
)
XE.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
XE1.local =(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1))
)`

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var results []*ldap.Entry
	var sslport int
	var tnsEntries TNSEntries
	var ldapTnsEntries TNSEntries
	var ldapCon *ldap.Conn

	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer destroyLdapContainer(ldapContainer)

	base := LdapBaseDn
	server, sslport = getLdapHostAndPort(ldapContainer, "636/tcp")
	ldaplib.SetConfig(server, sslport, true, true, base)
	context := ""
	t.Run("Ldap Connect", func(t *testing.T) {
		t.Logf("Connect '%s' using SSL on port %d", LdapAdminUser, sslport)
		ldapCon, err = ldaplib.Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, ldapCon, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, ldapCon, "returned object ist not ldap connection")
		if ldapCon == nil {
			t.Fatalf("No valid Connection, terminate")
			return
		}
	})

	t.Run("Get Oracle Context", func(t *testing.T) {
		context, err = GetOracleContext(ldapCon, base)
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
		tnsAdmin := TESTDATA
		filename := tnsAdmin + "/ldap_file.ora"
		//nolint gosec
		err = os.WriteFile(filename, []byte(ldaptns), 0644)
		require.NoErrorf(t, err, "Create test ldap_file.ora failed")
		t.Logf("load from %s", filename)

		// read entries from file
		tnsEntries, domain, err = GetTnsnames(filename, true)
		require.NoErrorf(t, err, "Parsing %s failed: %s", filename, err)
		if err != nil {
			t.Fatalf("tns load returned error: %s ", err)
			return
		}

		// write entries to ldap
		var workstatus TWorkStatus
		workstatus, err = WriteLdapTns(ldapCon, tnsEntries, domain, context)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
		expected := len(tnsEntries)
		actual := workstatus[sNew]
		require.Equal(t, expected, actual, "Not all Records has been added")
		t.Logf("%d Entries added", actual)
	})

	if err != nil {
		t.Fatalf("need Write TNS to proceed")
		return
	}
	t.Run("Ldap TNS Search", func(t *testing.T) {
		results, err = ldaplib.Search(ldapCon, context, "(objectclass=orclNetService)", []string{"DN"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		actual := len(results)
		assert.Greaterf(t, actual, 0, "Zero Entries")
		t.Logf("Returned %d entries", actual)
	})
	t.Run("Ldap TNS Read", func(t *testing.T) {
		ldapTnsEntries, err = ReadLdapTns(ldapCon, context)
		require.NoErrorf(t, err, "Ldap Read returned error:%v", err)
		actual := len(ldapTnsEntries)
		expected := len(tnsEntries)
		assert.Equal(t, expected, actual, "Entry Count differs")
	})
}
