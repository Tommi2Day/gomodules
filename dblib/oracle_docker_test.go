package dblib

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/ory/dockertest/v4"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/test"
)

// DBPort is the port of the Oracle DB to access
const DBPort = "21522"
const repo = "docker.io/gvenzl/oracle-free"
const repoTag = "23.26.2-slim"
const containerTimeout = 600

// SYSTEMUSER is the name of the default DBA user
const SYSTEMUSER = "system"

// SYSTEMSERVICE is the name of the root service
const SYSTEMSERVICE = "FREE"

var containerName string

// prepareContainer create an Oracle Docker Container
func prepareContainer() (resource dockertest.ClosableResource, err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}
	containerName = os.Getenv("CONTAINER_NAME")
	if containerName == "" {
		containerName = "dblib-oracledb"
	}
	var pool dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}
	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + repo

	ctx := context.Background()
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, repoTag)
	resource, err = pool.Run(ctx, repoString,
		dockertest.WithTag(repoTag),
		dockertest.WithHostname(containerName),
		dockertest.WithName(containerName),
		dockertest.WithEnv([]string{
			"ORACLE_PASSWORD=" + DBPASSWORD,
		}),
		// "./oracle:/container-entrypoint-initdb.d"
		dockertest.WithMounts([]string{
			test.TestDir + "/docker/oracle-db:/container-entrypoint-initdb.d:ro",
		}),
		dockertest.WithContainerConfig(func(config *container.Config) {
			if config.ExposedPorts == nil {
				config.ExposedPorts = network.PortSet{}
			}
			config.ExposedPorts[network.MustParsePort("1521/tcp")] = struct{}{}
		}),
		dockertest.WithHostConfig(func(config *container.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyDisabled}
			// need fixed mapping here
			config.PortBindings = network.PortMap{
				network.MustParsePort("1521/tcp"): {
					{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: DBPort},
				},
			}
		}),
	)

	if err != nil {
		err = fmt.Errorf("error starting DB docker %s container: %v", containerName, err)
		if resource != nil {
			_ = resource.Close(ctx)
		}
		return
	}
	err = WaitForOracle(pool)
	if err != nil {
		_ = resource.Close(ctx)
		return
	}
	return
}

// WaitForOracle waits to successfully connect to Oracle
func WaitForOracle(pool dockertest.Pool) (err error) {
	if os.Getenv("SKIP_ORACLE") != "" {
		err = fmt.Errorf("skipping ORACLE Container in CI environment")
		return
	}

	if pool == nil {
		pool, err = common.GetDockerPool()
		if err != nil {
			return
		}
	}

	target = fmt.Sprintf("oracle://%s:%s@%s:%s/%s", SYSTEMUSER, DBPASSWORD, DBhost, DBPort, SYSTEMSERVICE)
	fmt.Printf("Wait to successfully init db with %s (max %ds)...\n", target, containerTimeout)
	start := time.Now()
	if err = pool.Retry(context.Background(), containerTimeout*time.Second, func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("oracle", target)
		if err != nil {
			// cannot open connection
			return err
		}
		err = db.Ping()
		if err != nil {
			// db not answering
			return err
		}
		// check if init_done table exists, then we are ready
		checkSQL := "select count(*) from init_done"
		row := db.QueryRow(checkSQL)
		var count int64
		err = row.Scan(&count)
		if err != nil {
			// query failed, final init table not there
			return err
		}
		return nil
	}); err != nil {
		fmt.Printf("DB Container not ready: %s", err)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("DB Ready is available after %s\n", elapsed.Round(time.Millisecond))
	// wait for init scripts finished
	err = nil
	return
}
