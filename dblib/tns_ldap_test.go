package dblib

import (
	"fmt"
	"os"
	"testing"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/ldaplib"
	"github.com/tommi2day/gomodules/test"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const LdapBaseDn = "dc=oracle,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigUser = "cn=config"
const LdapConfigPassword = "config"

const ldaptns = `
(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE))
)
`

const ldaptns2 = `
(DESCRIPTION =
(ADDRESS_LIST = (ADDRESS=(PROTOCOL=TCP)(HOST=127.0.0.1)(PORT=1521)))
(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME = XE1.local))
)
`

const ldaporaOK = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (localhost:1389:1636, localhost:1389)
DIRECTORY_SERVER_TYPE = OID
`
const ldaporaFail = `
DEFAULT_ADMIN_CONTEXT = "dc=oracle,dc=local"
DIRECTORY_SERVERS = (localhost::1636, :389)
DIRECTORY_SERVER_TYPE = OID
`
const ldapTimeout = 20

func TestOracleLdap(t *testing.T) {
	var err error
	var server string
	var results []*ldap.Entry
	var port int
	var ldapTnsEntries TNSEntries
	var lc *ldaplib.LdapConfigType

	test.InitTestDirs()
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")
	ldapAdmin := test.TestData

	t.Run("Parse wrong ldap.ora", func(t *testing.T) {
		err = common.WriteStringToFile(ldapAdmin+"/ldap.ora", ldaporaFail)
		require.NoErrorf(t, err, "Create test ldap.ora failed")
		_, ldapservers := ReadLdapOra(ldapAdmin)
		e := 1
		l := len(ldapservers)
		assert.Equal(t, e, l, "ldapservers should have exact %d entries", 2)
		if l == e {
			s := ldapservers[0]
			expected := "localhost:0:1636"
			actual := fmt.Sprintf("%s:%d:%d", s.Hostname, s.Port, s.SSLPort)
			assert.Equal(t, expected, actual, "ldap entry 1 not match")
		}
	})

	err = common.WriteStringToFile(ldapAdmin+"/ldap.ora", ldaporaOK)
	require.NoErrorf(t, err, "Create test ldap.ora failed")
	t.Run("Parse ldap.ora", func(t *testing.T) {
		oraclecontext, ldapservers := ReadLdapOra(ldapAdmin)
		e := 2
		l := len(ldapservers)
		assert.NotEmpty(t, oraclecontext, "Context should not be empty")
		assert.Equal(t, e, l, "ldapservers should have exact %d entries", 2)
		if l == e {
			s := ldapservers[0]
			expected := "localhost:1389:1636"
			actual := fmt.Sprintf("%s:%d:%d", s.Hostname, s.Port, s.SSLPort)
			assert.Equal(t, expected, actual, "ldap entry 1 not match")
			s = ldapservers[1]
			expected = "localhost:1389:0"
			actual = fmt.Sprintf("%s:%d:%d", s.Hostname, s.Port, s.SSLPort)
			assert.Equal(t, expected, actual, "ldap entry 2 not match")
		}
	})

	// prepare or skip container based tests
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	TnsLdapContainer, err = prepareTnsLdapContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, TnsLdapContainer, "Prepare failed")
	defer common.DestroyDockerContainer(TnsLdapContainer)

	base := LdapBaseDn
	server, port = common.GetContainerHostAndPort(TnsLdapContainer, "389/tcp")
	lc = ldaplib.NewConfig(server, port, false, false, base, ldapTimeout)
	context := ""

	t.Run("Ldap Connect", func(t *testing.T) {
		t.Logf("Connect '%s' on port %d", LdapAdminUser, port)
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

	alias := "XE.local"
	t.Run("Add TNS Entry", func(t *testing.T) {
		err = os.Chdir(test.TestDir)
		require.NoErrorf(t, err, "ChDir failed")
		// write entries to ldap
		err = AddLdapTNSEntry(lc, context, alias, ldaptns)
		require.NoErrorf(t, err, "Write TNS to Ldap failed: %s", err)
	})
	// terminate if not succeeded
	if err != nil {
		t.Fatalf("need Write TNS to proceed")
		return
	}
	dn := ""
	t.Run("Ldap TNS Search", func(t *testing.T) {
		search := "(objectclass=orclNetService)"
		t.Logf("Search one level in %s for %s", context, search)
		results, err = lc.Search(context, search, []string{"DN"}, ldap.ScopeSingleLevel, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		actual := len(results)
		require.Greaterf(t, actual, 0, "Zero Entries")
		t.Logf("Returned %d entries", actual)
		dn = results[0].DN
		t.Logf("Entry-DN: %v", dn)
	})
	t.Run("Ldap TNS base query", func(t *testing.T) {
		search := fmt.Sprintf("cn=%s,%s", "XE.local", context)
		t.Logf("Search direct for %s as base", search)
		// direct entry searches uses DN as base and filer * with scope base
		results, err = lc.Search(search, "(objectClass=*)", []string{"DN"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		actual := len(results)
		require.Greaterf(t, actual, 0, "Zero Entries")
		t.Logf("Returned %d entries", actual)
		cn := results[0].GetEqualFoldAttributeValue("cn")
		desc := results[0].GetEqualFoldAttributeValue("orclNetDescString")
		t.Logf("%s=%s", cn, desc)
	})

	t.Run("Modify TNS Entry", func(t *testing.T) {
		ldapTnsEntries, err = ReadLdapTns(lc, context)
		require.NoErrorf(t, err, "Ldap Read returned error:%v", err)
		actual := len(ldapTnsEntries)
		expected := 1
		assert.Equal(t, expected, actual, "Entry Count differs")
		ldapTnsEntries, err = ReadLdapTns(lc, context)
		e, valid := ldapTnsEntries[alias]
		require.Truef(t, valid, "Entry not found")
		dn = e.Location
		err = ModifyLdapTNSEntry(lc, dn, alias, ldaptns2)
		require.NoErrorf(t, err, "Modify Ldap failed: %s", err)
	})
	t.Run("Delete TNS Entry", func(t *testing.T) {
		err = DeleteLdapTNSEntry(lc, dn, alias)
		require.NoErrorf(t, err, "Delete TNS from Ldap failed: %s", err)
	})
}
