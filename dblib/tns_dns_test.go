package dblib

import (
	"fmt"
	"os"
	"testing"

	"github.com/tommi2day/gomodules/netlib"

	log "github.com/sirupsen/logrus"

	"github.com/tommi2day/gomodules/common"

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

func TestMain(m *testing.M) {
	var err error

	test.InitTestDirs()
	if os.Getenv("SKIP_DB_DNS") != "" {
		fmt.Println("Skipping DB DNS Container in CI environment")
		return
	}
	dblibDNSServer = common.GetStringEnv("DNS_HOST", dblibDNSServer)
	dblibDNSContainer, err = prepareDBlibDNSContainer()

	if err != nil || dblibDNSContainer == nil {
		_ = os.Setenv("SKIP_DB_DNS", "true")
		log.Errorf("prepareNetlibDNSContainer failed: %s", err)
		destroyDNSContainer(dblibDNSContainer)
	}

	code := m.Run()
	destroyDNSContainer(dblibDNSContainer)
	os.Exit(code)
}

func TestRACInfo(t *testing.T) {
	var err error
	test.InitTestDirs()
	err = os.Chdir(test.TestDir)
	require.NoErrorf(t, err, "ChDir failed")

	//nolint gosec
	err = common.WriteStringToFile(tnsAdmin+"/racinfo.ini", racinfoini)
	require.NoErrorf(t, err, "Create test racinfo.ini failed")

	if os.Getenv("SKIP_DB_DNS") != "" {
		t.Skip("Skipping DB DNS testing")
	}
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
