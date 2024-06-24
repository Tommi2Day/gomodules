package dblib

import (
	"os"
	"testing"

	"github.com/tommi2day/gomodules/netlib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tommi2day/gomodules/test"
)

const racaddr = "myrac.rac.lan"
const racinfoini = `
[MYRAC.RAC.LAN]
scan=myrac.rac.lan:1521
vip1=vip1.rac.lan:1521
vip2=vip2.rac.lan:1521
vip3=vip3.rac.lan:1521
`

func TestRACInfo(t *testing.T) {
	var err error
	test.InitTestDirs()
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	//nolint gosec
	err = os.WriteFile(tnsAdmin+"/racinfo.ini", []byte(racinfoini), 0644)
	require.NoErrorf(t, err, "Create test racinfo.ini failed")

	if os.Getenv("SKIP_DNS") != "" {
		t.Skip("Skipping DNS testing in CI environment")
	}
	dblibDNSContainer, err = prepareDNSContainer()
	require.NoErrorf(t, err, "DNS Server not available")
	require.NotNil(t, dblibDNSContainer, "Prepare failed")
	defer destroyDNSContainer(dblibDNSContainer)
	// use DNS from Docker
	DNSConfig = netlib.NewResolver(dblibDNSServer, dblibDNSPort, true)
	t.Run("Test RacInfo.ini resolution", func(t *testing.T) {
		IgnoreDNSLookup = false
		IPv4Only = true
		addr := GetRacAdresses(racaddr, tnsAdmin+"/racinfo.ini")
		assert.Equal(t, 6, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
	t.Run("Test Rac SRV resolution", func(t *testing.T) {
		IPv4Only = true
		IgnoreDNSLookup = false
		addr := GetRacAdresses(racaddr, "")
		assert.Equal(t, 6, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
	t.Run("Test resolution with IP", func(t *testing.T) {
		IPv4Only = true
		IgnoreDNSLookup = false
		addr := GetRacAdresses("127.0.0.1", "")
		assert.Equal(t, 0, len(addr), "Count not expected")
		t.Logf("Addresses: %v", addr)
	})
}
