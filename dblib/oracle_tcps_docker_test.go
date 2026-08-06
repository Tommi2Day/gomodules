package dblib

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"os"
	"path"
	"testing"
	"time"

	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"

	ora "github.com/sijms/go-ora/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/ory/dockertest/v4"
)

// TCPS testing needs a real Oracle wallet (orapki) which requires the "-full"
// image; the "-slim" image used by the other Oracle tests has no Java and
// therefore no working orapki/mkstore. The "-faststart" variant restores from
// a pre-built snapshot instead of running dbca from scratch, which matters
// here since the plain "-full" tag can be too slow in CI to finish within
// go test's default 10-minute binary-wide deadline.
const (
	tcpsRepoTag = "23.26.2-full-faststart"
	tcpsDBPort  = "21525"
	tcpsPort    = "21526"
	// keep this safely under go test's default 10-minute (600s) overall
	// deadline (not equal to it - that races the test binary's own alarm)
	// so a genuinely stuck container fails with a clear message here instead
	// of an opaque timeout panic that doesn't print our last error
	tcpsContainerTimeout = 480
)

var tcpsContainerName string

// prepareTCPSContainer starts an Oracle "-full" container whose
// container-entrypoint-initdb.d scripts (test/docker/oracle-db-tcps) create a
// real orapki auto-login wallet and configure a TCPS listener with it, then
// copy that wallet into walletDir (bind-mounted at /client-wallet) so the
// test can use it as WALLET_LOCATION exactly like a real client would.
func prepareTCPSContainer(walletDir string) (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	tcpsContainerName = os.Getenv("TCPS_CONTAINER_NAME")
	if tcpsContainerName == "" {
		tcpsContainerName = "dblib-oracledb-tcps"
	}
	var pool dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + repo

	ctx := context.Background()
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, tcpsRepoTag)
	resource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(tcpsRepoTag),
		dockertest.WithHostname(tcpsContainerName),
		dockertest.WithName(tcpsContainerName),
		dockertest.WithEnv([]string{
			"ORACLE_PASSWORD=" + DBPASSWORD,
		}),
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/oracle-db-tcps:/container-entrypoint-initdb.d:ro",
			walletDir + ":/client-wallet",
		}),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("1521/tcp")] = struct{}{}
			config.ExposedPorts[network.MustParsePort("2484/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			config.PortBindings = network.PortMap{
				network.MustParsePort("1521/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: tcpsDBPort},
				},
				network.MustParsePort("2484/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: tcpsPort},
				},
			}
		}),
	)
	if err != nil {
		err = fmt.Errorf("error starting TCPS DB docker %s container: %v", tcpsContainerName, err)
		if resource != nil {
			_ = resource.Close(ctx)
		}
		return
	}

	target := fmt.Sprintf("oracle://%s:%s@%s:%s/%s", SYSTEMUSER, DBPASSWORD, DBhost, tcpsDBPort, SYSTEMSERVICE)
	fmt.Printf("Wait to successfully init TCPS db with %s (max %ds)...\n", target, tcpsContainerTimeout)
	start := time.Now()
	lastLog := start
	if err = pool.Retry(ctx, tcpsContainerTimeout*time.Second, func() error {
		attemptErr := func() error {
			db, dbErr := sql.Open("oracle", target)
			if dbErr != nil {
				return dbErr
			}
			if dbErr = db.Ping(); dbErr != nil {
				return dbErr
			}
			// init_done only exists once every init script, including the
			// TCPS wallet/listener setup, has finished
			row := db.QueryRow("select count(*) from init_done")
			var count int64
			return row.Scan(&count)
		}()
		// this can silently loop for minutes; log the last error periodically
		// so a stuck run is diagnosable instead of a bare test-timeout panic
		if attemptErr != nil && time.Since(lastLog) > 30*time.Second {
			fmt.Printf("TCPS DB still not ready after %s: %v\n", time.Since(start).Round(time.Second), attemptErr)
			lastLog = time.Now()
		}
		return attemptErr
	}); err != nil {
		fmt.Printf("TCPS DB Container not ready: %s\n", err)
		_ = resource.Close(ctx)
		return
	}
	fmt.Printf("TCPS DB Ready is available after %s\n", time.Since(start).Round(time.Millisecond))
	return
}

// TestOracleTCPSConnect verifies a real connect through a real TCPS listener
// using a real orapki auto-login wallet, the same way sqlplus would: TNS
// entry with PROTOCOL=TCPS plus WALLET_LOCATION in sqlnet.ora, no password
// needed since the wallet is auto-login (matches production practice).
func TestOracleTCPSConnect(t *testing.T) {
	if os.Getenv("SKIP_ORACLE") != "" {
		t.Skip("Skipping ORACLE testing in CI environment")
	}
	test.InitTestDirs()

	// TNSSSLconfig is a shared package var; make sure this test doesn't
	// leak wallet/SSL state into tests that run after it.
	defer func() { TNSSSLconfig = TNSSSL{} }()

	tcpsDir := path.Join(test.TestData, "tcps")
	walletDir := path.Join(tcpsDir, "wallet")
	// clean up any stale wallet from a previous failed run before recreating
	_ = os.RemoveAll(walletDir)
	require.NoErrorf(t, os.MkdirAll(walletDir, 0777), "create wallet dir failed")
	// MkdirAll's mode is masked by umask; chmod explicitly so the
	// container's oracle user (a different uid) can write the wallet here
	require.NoErrorf(t, os.Chmod(walletDir, 0777), "chmod wallet dir failed")

	sqlnetContent := fmt.Sprintf("WALLET_LOCATION=(SOURCE=(METHOD=FILE)(METHOD_DATA=(DIRECTORY=\"%s\")))\n", walletDir)
	require.NoErrorf(t, common.WriteStringToFile(path.Join(tcpsDir, "sqlnet.ora"), sqlnetContent), "write sqlnet.ora failed")

	tcpsAlias := "TCPSTEST.local"
	tcpsDesc := fmt.Sprintf("(DESCRIPTION=(ADDRESS_LIST=(ADDRESS=(PROTOCOL=TCPS)(HOST=%s)(PORT=%s)))(CONNECT_DATA=(SERVER=DEDICATED)(SERVICE_NAME=%s)))", DBhost, tcpsPort, DBPDB)
	tnsFilename := path.Join(tcpsDir, "tcps_connect.ora")
	require.NoErrorf(t, common.WriteStringToFile(tnsFilename, tcpsAlias+"="+tcpsDesc), "write tnsnames failed")

	oracleContainer, err := prepareTCPSContainer(walletDir)
	require.NoErrorf(t, err, "prepare TCPS Oracle Container failed")
	require.NotNil(t, oracleContainer, "Prepare failed")
	defer common.DestroyDockerContainer(oracleContainer)

	tnsEntries, domain, err := GetTnsnames(tnsFilename, true)
	require.NoErrorf(t, err, "Parsing %s failed: %s", tnsFilename, err)
	e, found := GetEntry(tcpsAlias, tnsEntries, domain)
	require.True(t, found, "Alias not found")
	desc := common.RemoveSpace(e.Desc)

	t.Run("Check TCPS without wallet fails", func(t *testing.T) {
		TNSSSLconfig = TNSSSL{}
		options := SSLConnectOptions(desc)
		connect := ora.BuildJDBC(SYSTEMUSER, DBPASSWORD, desc, options)
		_, connErr := DBConnect("oracle", connect, TIMEOUT)
		assert.Error(t, connErr, "connect without wallet should fail")
	})

	t.Run("Check TCPS with wallet succeeds", func(t *testing.T) {
		LoadSSLConfig(tcpsDir)
		require.Equal(t, walletDir, TNSSSLconfig.WalletLocation, "wallet location not resolved as expected")
		options := SSLConnectOptions(desc)
		connect := ora.BuildJDBC(SYSTEMUSER, DBPASSWORD, desc, options)
		db, connErr := DBConnect("oracle", connect, TIMEOUT)
		require.NoErrorf(t, connErr, "TCPS connect with wallet should succeed: %v", connErr)
		result, selErr := SelectOneStringValue(db, "select to_char(sysdate,'YYYY-MM-DD HH24:MI:SS') from dual")
		assert.NoErrorf(t, selErr, "query over TCPS failed: %v", selErr)
		assert.NotEmpty(t, result)
		t.Logf("Sysdate over TCPS: %s", result)
	})
}
