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
		SetConfig("ldap.test", 0, true, true, LdapBaseDn)
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
	var entries []*ldap.Entry
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}

	ldapContainer, err = prepareContainer()
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	defer destroyContainer(ldapContainer)

	server, port = getHostAndPort(ldapContainer, "389/tcp")
	base := LdapBaseDn
	SetConfig(server, port, false, false, LdapBaseDn)
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
	SetConfig(server, sslport, true, true, LdapBaseDn)
	t.Run("Admin SSL Connect", func(t *testing.T) {
		t.Logf("Connect Admin '%s' using SSL on port %d", LdapAdminUser, sslport)
		l, err = Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, l, "returned object ist not ldap connection")
	})
	t.Run("BaseDN Search", func(t *testing.T) {
		entries, err = Search(l, base, "(objectclass=*)", []string{"DN"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "Search returned error:%v", err)
		count := len(entries)
		assert.Greaterf(t, count, 0, "Zero Entries")
		if count > 0 {
			dn := entries[0].DN
			t.Logf("Base DN: %s", dn)
			assert.Equal(t, base, dn, "DN not equal to base")
		}
	})
	userDN := "cn=testuser," + LdapBaseDn
	userPass := "testPass"

	t.Run("Add Entry", func(t *testing.T) {
		var Attributes []ldap.Attribute
		Attributes = append(Attributes, ldap.Attribute{Type: "objectClass", Vals: []string{"top", "iNetOrgPerson"}})
		Attributes = append(Attributes, ldap.Attribute{Type: "cn", Vals: []string{"testuser"}})
		Attributes = append(Attributes, ldap.Attribute{Type: "sn", Vals: []string{"User"}})
		Attributes = append(Attributes, ldap.Attribute{Type: "gn", Vals: []string{"Test"}})
		// Attributes =append(Attributes,ldap.Attribute{Type: "uid", Vals: []string{ "c666f5ab-1b26-4421-9eb1-50775ed96hf6" }})
		Attributes = append(Attributes, ldap.Attribute{Type: "mail", Vals: []string{"testuser@" + LdapDomain}})
		//Attributes = append(Attributes, ldap.Attribute{Type: "userPassword", Vals: []string{"{crypt}x"}})
		err = AddEntry(l, userDN, Attributes)
		assert.NoErrorf(t, err, "Add User failed")
		_, err = SetPassword(l, userDN, "", userPass)
		require.NoErrorf(t, err, "Test Bind fix Pass returned error %v", err)
	})
	t.Run("Modify Attribute", func(t *testing.T) {
		newMail := "testmail@test.com"
		err = ModifyAttribute(l, userDN, "modify", "mail", []string{newMail})
		require.NoErrorf(t, err, "Entry  mail was not modified and returned error %v", err)
		// test change
		entries, err = Search(l, userDN, "(objectclass=*)", []string{"DN", "mail"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		require.Equalf(t, 1, len(entries), "Should return only one entry")
		actMail := entries[0].GetAttributeValue("mail")
		assert.Equal(t, newMail, actMail, "Mail Modify not visible")
	})
	t.Run("Delete Attribute", func(t *testing.T) {
		err = ModifyAttribute(l, userDN, "delete", "gn", nil)
		// test change
		entries, err = Search(l, userDN, "(objectclass=*)", []string{"DN", "gn"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		require.Equalf(t, 1, len(entries), "Should return only one entry")
		actAttr := entries[0].GetAttributeValue("gn")
		assert.Emptyf(t, actAttr, "Attribute gn should not exists")
	})
	t.Run("Change User Password", func(t *testing.T) {
		var genPass string
		// connect to testuser with new pass
		l, err = Connect(userDN, userPass)
		require.NoErrorf(t, err, "Test Bind with new password returned error %v", err)
		genPass, err = SetPassword(l, "", userPass, "")
		require.NoErrorf(t, err, "Generate Password returned Error: %v", err)
		assert.NotEmptyf(t, genPass, "no password was generated")
		t.Logf("generated Password: %s", genPass)
		l.Close()

		//reconnect with new password
		l, err = Connect(userDN, genPass)
		assert.NoErrorf(t, err, "Test Bind with generated password returned error %v", err)
		if l != nil {
			l.Close()
		}
	})

	t.Run("Delete Entry", func(t *testing.T) {
		l, err = Connect(LdapAdminUser, LdapAdminPassword)
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		err = DeleteEntry(l, userDN)
		assert.NoErrorf(t, err, "Deleting failed")
		entries, err = Search(l, userDN, "(objectclass=*)", []string{"DN"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		assert.NoErrorf(t, err, "Should not return any error as no data error was removed")
		assert.Equalf(t, 0, len(entries), "Should return no one entry")
	})
}
