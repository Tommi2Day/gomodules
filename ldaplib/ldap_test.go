package ldaplib

import (
	"os"
	"testing"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ldapOrganisation = "Ldap Test Ltd"
const LdapDomain = "ldap.test"
const LdapBaseDn = "dc=ldap,dc=test"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigPassword = "config"

var port = 10389
var sslport = 10636

func TestLdapConfig(t *testing.T) {
	t.Run("Ldap Config", func(t *testing.T) {
		SetConfig("ldap.test", 0, true, true)
		actual := GetConfig()
		assert.IsType(t, ConfigType{}, actual, "Wrong type returned")
		assert.Equal(t, "ldap.test", actual.Server, "Server not equal")
		assert.Equal(t, 636, actual.Port, "with tls=true port should be 636")
		assert.Equal(t, "ldaps://ldap.test:636", actual.URL, "with tls=true should be ldaps")
	})
}

func TestBaseLdap(t *testing.T) {
	var l *ldap.Conn
	var err error
	var server string
	var results *ldap.SearchResult
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	ldapContainer, err = prepareContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer destroyContainer(ldapContainer)

	server, port = getHostAndPort(ldapContainer, "389/tcp")
	base := LdapBaseDn
	SetConfig(server, port, false, false)
	t.Run("Anonymous Connect", func(t *testing.T) {
		t.Logf("Connect anonymous plain on port %d", port)
		l, err = Connect("", "")
		require.NoErrorf(t, err, "anonymous Connect returned error: %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, l, "returned object ist not ldap connection")
		l.Close()
	})
	// test container should not be validaed
	server, sslport = getHostAndPort(ldapContainer, "636/tcp")
	SetConfig(server, sslport, true, true)
	t.Run("Admin SSL Connect", func(t *testing.T) {
		t.Logf("Connect Admin '%s' using SSL on port %d", LdapAdminUser, sslport)
		l, err = Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, l, "returned object ist not ldap connection")
	})
	t.Run("BaseDN Search", func(t *testing.T) {
		results, err = Search(l, base, "(objectclass=*)", []string{"DN"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		count := len(results.Entries)
		assert.Greaterf(t, count, 0, "Zero Entries")
		if count > 0 {
			dn := results.Entries[0].DN
			t.Logf("Base DN: %s", dn)
			assert.Equal(t, base, dn, "DN not equal to base")
		}
	})
}
