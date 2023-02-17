package ldap

import (
	goldap "github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestLdapConfig(t *testing.T) {
	t.Run("Ldap Config", func(t *testing.T) {
		SetConfig("ldap.test", 0, true)
		actual := GetConfig()
		assert.IsType(t, ConfigType{}, actual, "Wrong type returned")
		assert.Equal(t, "ldap.test", actual.Server, "Server not equal")
		assert.Equal(t, 636, actual.Port, "with tls=true port should be 636")
	})
}
func TestBaseLdap(t *testing.T) {
	var l *goldap.Conn
	var err error
	var results *goldap.SearchResult
	if os.Getenv("SKIP_LDAP") != "" {
		t.Skip("Skipping Mail testing in CI environment")
	}
	server := os.Getenv("LDAP_SERVER")
	base := os.Getenv("LDAP_BASEDN")
	if len(server) == 0 {
		server = "localhost"
	}
	if len(base) == 0 {
		base = ""
	}
	SetConfig(server, 389, false)
	t.Run("Anonym Connect", func(t *testing.T) {
		l, err = Connect("", "")
		require.NoErrorf(t, err, "anonymous Connect returned error %v", err)
		assert.NotNilf(t, l, "Ldap Connect is nil")
		assert.IsType(t, &goldap.Conn{}, l, "returned object ist not ldap connection")
	})
	t.Run("Search", func(t *testing.T) {
		results, err = Search(l, base, "(objectclass=*)", []string{"DN"}, goldap.ScopeBaseObject, goldap.DerefInSearching)
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
