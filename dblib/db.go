// Package dblib collection of db func
package dblib

import (
	"context"
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
)

// https://www.alexedwards.net/blog/how-to-manage-database-timeouts-and-cancellations-in-go
// https://github.com/sijms/go-ora/#version-241-add-support-for-connection-time-out--context-read-and-write

// DBConnect connect to a database using connect string
func DBConnect(driver string, source string, timeout int) (db *sql.DB, err error) {
	const defaultTimeout = 5
	// Create a new child context with a 5-second timeout, using the
	// provided ctx parameter as the parent.

	if timeout < defaultTimeout {
		timeout = defaultTimeout
	}
	log.Debugf("try to connect, timeout %d", timeout)
	db, err = sql.Open(driver, source)
	if err != nil {
		return nil, err
	}

	// Create a context with timeout, using the empty
	// context.Background() as the parent.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Use this when testing the connection pool.
	if err = db.PingContext(ctx); err != nil {
		log.Debugf("DB Connect returned error= %s", err)
		return nil, err
	}
	log.Debugf("DB Connect success")
	return db, err
}

// SelectOneStringValue Select a single string
func SelectOneStringValue(db *sql.DB, sql string) (queryResult string, err error) {
	row := db.QueryRow(sql)
	err = row.Scan(&queryResult)
	if err != nil {
		if isOerr, _, msg := HaveOerr(err); isOerr {
			log.Warnf("Oracle Error %s", msg)
		} else {
			log.Warnf("got error %s", err)
		}
	}
	return
}
