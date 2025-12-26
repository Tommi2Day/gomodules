package ldaplib

import (
	"os"
	"testing"

	"github.com/tommi2day/gomodules/test"

	"github.com/tommi2day/gomodules/common"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const LdapDomain = "example.local"
const LdapBaseDn = "dc=example,dc=local"
const LdapAdminUser = "cn=admin," + LdapBaseDn
const LdapAdminPassword = "admin"
const LdapConfigUser = "cn=config"
const LdapConfigPassword = "config"

// const LdapOrganisation = "Example Org"

const testLdif = `
dn: ou=Test,dc=example,dc=local
ou: Groups
objectClass: top
objectClass: organizationalUnit
`
const errorLdif = `
dn: ou=Error,dc=example,dc=local
ou: Groups
`

var ldapPort int
var sslport int
var lc *LdapConfigType
var timeout = 20

func TestLdapConfig(t *testing.T) {
	t.Run("Ldap Config", func(t *testing.T) {
		lc = NewConfig("ldap.test", 0, true, true, LdapBaseDn, timeout)
		actual := lc
		assert.Equal(t, "ldap.test", actual.Server, "Server not equal")
		assert.Equal(t, 636, actual.Port, "with tls=true ldapPort should be 636")
		assert.Equal(t, "ldaps://ldap.test:636", actual.URL, "with tls=true should be ldaps")
	})
}

// nolint gocognit
func TestBaseLdap(t *testing.T) {
	var l *ldap.Conn
	var err error
	var ldapserver string
	var entries []*ldap.Entry
	var entry *ldap.Entry
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping LDAP testing in CI environment")
	}
	// configLdif = strings.ReplaceAll(configLdif, "%BASE%", LdapBaseDn)
	test.InitTestDirs()
	ldapContainer, err = prepareLdapContainer()
	defer common.DestroyDockerContainer(ldapContainer)
	require.NoErrorf(t, err, "Ldap Server not available")
	require.NotNil(t, ldapContainer, "Prepare failed")
	if err != nil || ldapContainer == nil {
		t.Fatal("LDAP server not available")
	}

	ldapserver, ldapPort = common.GetContainerHostAndPort(ldapContainer, "389/tcp")
	base := LdapBaseDn
	ldifDir := test.TestDir + "/docker/ldap/ldif"
	lc = NewConfig(ldapserver, ldapPort, false, false, LdapBaseDn, timeout)
	t.Run("Anonymous Connect", func(t *testing.T) {
		t.Logf("Connect anonymous plain on ldapPort %d", ldapPort)
		err = lc.Connect("", "")
		l = lc.Conn
		require.NoErrorf(t, err, "anonymous Connect returned error: %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, l, "returned object ist not ldap connection")
		_ = l.Close()
	})

	t.Run("Apply LDIF Config from Directory", func(t *testing.T) {
		err = lc.Connect(LdapConfigUser, LdapConfigPassword)
		require.NoErrorf(t, err, "Config Connect failed: %v", err)

		// Path inside the project for testing
		pattern := "*.schema"
		// Apply all files matching *.config
		err = lc.ApplyLDIFDir(ldifDir, pattern, false)
		assert.NoErrorf(t, err, "Apply schemas %s/%s failed: %v", ldifDir, pattern, err)

		// Verify by searching for one of the applied schemas/configs if needed
		// For example, checking if a specific schema DN exists
		schemaBase := "cn=schema,cn=config"
		entries, err = lc.Search(schemaBase, "(cn=*ldapPublicKey)", []string{"dn"}, ldap.ScopeWholeSubtree, ldap.DerefInSearching)
		assert.NoErrorf(t, err, "Search for schema ldapPublicKey failed: %v", err)
		assert.NotNil(t, entries, "Search returned nil")
		assert.Greaterf(t, len(entries), 0, "no entries found")
		if err == nil && len(entries) > 0 {
			t.Logf("Schema Verified: %s exists", entries[0].DN)
		}
		pattern = "*.config"
		err = lc.ApplyLDIFDir(ldifDir, pattern, false)
		require.NoErrorf(t, err, "Apply configs %s/%s failed: %v", ldifDir, pattern, err)
	})
	// test ssl connection
	ldapserver, sslport = common.GetContainerHostAndPort(ldapContainer, "636/tcp")
	lc = NewConfig(ldapserver, sslport, true, true, LdapBaseDn, timeout)
	t.Run("Admin SSL Connect", func(t *testing.T) {
		t.Logf("Connect Admin '%s' using SSL on ldapPort %d", LdapAdminUser, sslport)
		err = lc.Connect(LdapAdminUser, LdapAdminPassword)
		l = lc.Conn
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &ldap.Conn{}, l, "returned object ist not ldap connection")
	})
	if lc.Conn == nil {
		t.Fatal("Ldap Connection is nil")
	}
	t.Run("Add Base Entries", func(t *testing.T) {
		pattern := "*.ldif"
		err = lc.ApplyLDIFDir(ldifDir, pattern, false)
		require.NoErrorf(t, err, "Add Base Entries failed: %v", err)
	})

	t.Run("BaseDN Search", func(t *testing.T) {
		entry, err = lc.RetrieveEntry(base, "", "DN")
		require.NoErrorf(t, err, "Search returned error:%v", err)
		require.NotNil(t, entry, "Should return vald entry")
		if entry != nil {
			dn := entry.DN
			t.Logf("Base DN: %s", dn)
			assert.Equal(t, base, dn, "DN not equal to base")
		}
	})
	userDN := "cn=testuser," + LdapBaseDn
	userPass := "testPass"

	t.Run("Ldif Apply from String", func(t *testing.T) {
		dn := "ou=Test,dc=example,dc=local"
		err = lc.ApplyLDIF(testLdif, false)
		require.NoErrorf(t, err, "Apply returned error:%v", err)
		entry, err = lc.RetrieveEntry(dn, "", "DN")
		require.NoErrorf(t, err, "Search returned error:%v", err)
		require.NotNil(t, entry, "Should return valid entry")
		if entry != nil {
			actual := entry.DN
			t.Logf("Test DN: %s", dn)
			assert.Equal(t, dn, actual, "DN not equal to base")
		}
	})
	t.Run("Ldif Apply Error", func(t *testing.T) {
		dn := "ou=Error,dc=example,dc=local"
		err = lc.ApplyLDIF(errorLdif, false)
		require.Error(t, err, "Apply should return an error", err)
		if err != nil {
			t.Logf("Error: %v", err)
		}
		entry, err = lc.RetrieveEntry(dn, "", "DN")
		require.Nil(t, entry, "Should not return valid entry")
		if entry != nil {
			actual := entry.DN
			t.Logf("Error DN: %s", dn)
			assert.Equal(t, dn, actual, "DN not equal to base")
		}
	})

	t.Run("Add Entry", func(t *testing.T) {
		var attributes []ldap.Attribute
		attributes = append(attributes, ldap.Attribute{Type: "objectClass", Vals: []string{"top", "iNetOrgPerson"}})
		attributes = append(attributes, ldap.Attribute{Type: "cn", Vals: []string{"testuser"}})
		attributes = append(attributes, ldap.Attribute{Type: "sn", Vals: []string{"User"}})
		attributes = append(attributes, ldap.Attribute{Type: "gn", Vals: []string{"Test"}})
		attributes = append(attributes, ldap.Attribute{Type: "mail", Vals: []string{"testuser@" + LdapDomain}})

		err = lc.AddEntry(userDN, attributes)
		assert.NoErrorf(t, err, "Add User failed")
		_, err = lc.SetPassword(userDN, "", userPass)
		require.NoErrorf(t, err, "Test Bind fix Pass returned error %v", err)
	})
	t.Run("Export Entry", func(t *testing.T) {
		entries = []*ldap.Entry{}
		entry, err = lc.RetrieveEntry(userDN, "", "*")
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		assert.NotNilf(t, entry, "Should return vald entry")
		if entry != nil {
			entries = append(entries, entry)
			ldifData, err := ExportLDIF(entries)
			require.NoErrorf(t, err, "Export failed")
			assert.Containsf(t, ldifData, userDN, "Exported DN not equal to searched DN")
			t.Logf("LDIF Export Data: %s", ldifData)
		}
	})
	t.Run("Test HasObjectclass", func(t *testing.T) {
		entry, err = lc.RetrieveEntry(userDN, "", "DN,objectclass")
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		hasClass := HasObjectClass(entry, "iNetOrgPerson")
		assert.Truef(t, hasClass, "Objectclass iNetOrgPerson should exists")
		hasClass = HasObjectClass(entry, "ldapPublicKey")
		assert.False(t, hasClass, "Objectclass ldapPublicKey should not exists")
	})
	t.Run("Test HasAttribute", func(t *testing.T) {
		entry, err = lc.RetrieveEntry(userDN, "", "*")
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		hasAttr := HasAttribute(entry, "mail")
		assert.Truef(t, hasAttr, "Attribute mail should exists")
		hasAttr = HasAttribute(entry, "sshPublicKey")
		assert.False(t, hasAttr, "Attribute sshPublic should not exists")
	})
	t.Run("Modify Attribute", func(t *testing.T) {
		newMail := "testmail@test.com"
		err = lc.ModifyAttribute(userDN, "modify", "mail", []string{newMail})
		require.NoErrorf(t, err, "Entry  mail was not modified and returned error %v", err)
		// test change
		entries, err = lc.Search(userDN, "(objectclass=*)", []string{"DN", "mail"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		require.Equalf(t, 1, len(entries), "Should return only one entry")
		actMail := entries[0].GetAttributeValue("mail")
		assert.Equal(t, newMail, actMail, "Mail Modify not visible")
	})
	t.Run("Delete Attribute", func(t *testing.T) {
		err = lc.ModifyAttribute(userDN, "delete", "gn", nil)
		// test change
		entries, err = lc.Search(userDN, "(objectclass=*)", []string{"DN", "gn"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		require.NoErrorf(t, err, "search for %s returned error %v", userDN, err)
		require.Equalf(t, 1, len(entries), "Should return only one entry")
		actAttr := entries[0].GetAttributeValue("gn")
		assert.Emptyf(t, actAttr, "Attribute gn should not exists")
	})
	t.Run("Change User Password", func(t *testing.T) {
		var genPass string
		// connect to testuser with new pass
		err = lc.Connect(userDN, userPass)
		require.NoErrorf(t, err, "Test Bind with new password returned error %v", err)
		genPass, err = lc.SetPassword("", userPass, "")
		require.NoErrorf(t, err, "Generate Password returned Error: %v", err)
		assert.NotEmptyf(t, genPass, "no password was generated")
		t.Logf("generated Password: %s", genPass)
		_ = l.Close()

		// reconnect with new password
		err = lc.Connect(userDN, genPass)
		l = lc.Conn
		assert.NoErrorf(t, err, "Test Bind with generated password returned error %v", err)
		if l != nil {
			_ = l.Close()
		}
	})

	t.Run("Delete Entry", func(t *testing.T) {
		err = lc.Connect(LdapAdminUser, LdapAdminPassword)
		l = lc.Conn
		require.NoErrorf(t, err, "admin Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		err = lc.DeleteEntry(userDN)
		assert.NoErrorf(t, err, "Deleting failed")

		// check if we can find the dropped DN
		entries, err = lc.Search(userDN, "(objectclass=*)", []string{"DN"}, ldap.ScopeBaseObject, ldap.DerefInSearching)
		assert.NoErrorf(t, err, "Should not return any error as no data error was removed")
		assert.Equalf(t, 0, len(entries), "Should return no one entry")
	})
}
