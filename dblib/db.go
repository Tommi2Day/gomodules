// Package dblib collection of db func
package dblib

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/tommi2day/gomodules/common"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// https://www.alexedwards.net/blog/how-to-manage-database-timeouts-and-cancellations-in-go
// https://github.com/sijms/go-ora/#version-241-add-support-for-connection-time-out--context-read-and-write

// DBConnect connect to a database using connect string
func DBConnect(driver string, source string, timeout int) (dbh *sqlx.DB, err error) {
	const defaultTimeout = 5
	// Create a new child context with a 5-second timeout, using the
	// provided ctx parameter as the parent.

	if timeout < defaultTimeout {
		timeout = defaultTimeout
	}
	log.Debugf("try to connect, timeout %d", timeout)
	dbh, err = sqlx.Open(driver, source)
	if err != nil {
		return nil, err
	}

	// Create a context with timeout, using the empty
	// context.Background() as the parent.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Use this when testing the connection pool.
	if err = dbh.PingContext(ctx); err != nil {
		log.Debugf("DB Connect returned error= %s", err)
		return nil, err
	}
	log.Debugf("DB Connect success")
	return dbh, err
}

// SelectOneStringValue Select a single string
func SelectOneStringValue(dbh *sqlx.DB, mySQL string, args ...any) (resultString string, err error) {
	log.Debugf("SelectOneStringValue entered")
	err = SelectOneRow(dbh, mySQL, &resultString, args...)
	return
}

// SelectOneInt64Value Select a int64 string
func SelectOneInt64Value(dbh *sqlx.DB, mySQL string, args ...any) (resultInt int64, err error) {
	log.Debugf("SelectOneStringValue entered")
	err = SelectOneRow(dbh, mySQL, &resultInt, args...)
	return
}

// PrepareSQL parses a sql for a given connection and returns statement handler
func PrepareSQL(dbh *sqlx.DB, mySQL string) (stmt *sqlx.Stmt, err error) {
	log.Debugf("PrepareSql entered")
	ok, t := common.CheckType(dbh, "*sqlx.DB")
	if !ok {
		err = fmt.Errorf("invalid dbh %s", t)
		return
	}
	stmt, err = dbh.Preparex(mySQL)
	if err != nil {
		err = fmt.Errorf("prepare failed:%s (%s)", err, mySQL)
	}
	return
}

// PrepareSQLTx parses a sql for a given transaction and returns statement handler
func PrepareSQLTx(tx *sqlx.Tx, mySQL string) (stmt *sqlx.Stmt, err error) {
	log.Debugf("PrepareSql entered")
	ok, t := common.CheckType(tx, "*sqlx.Tx")
	if !ok {
		err = fmt.Errorf("invalid transaction %s", t)
		return
	}
	stmt, err = tx.Preparex(mySQL)
	if err != nil {
		err = fmt.Errorf("prepare failed:%s (%s)", err, mySQL)
	}
	return
}

// SelectOneRow combines preparing a statement, add bind variables and returns results
func SelectOneRow(dbh *sqlx.DB, mySQL string, result interface{}, args ...any) (err error) {
	log.Debugf("QueryOnRow entered")
	ok, t := common.CheckType(dbh, "*sqlx.DB")
	if !ok {
		err = fmt.Errorf("invalid dbh %s", t)
		return
	}
	log.Debugf("SQL: %s", mySQL)
	err = dbh.Get(result, mySQL, args...)
	return
}

// SelectAllRows combines preparing a statement, add bind variables and returns results struct
func SelectAllRows(dbh *sqlx.DB, mySQL string, result interface{}, args ...any) (err error) {
	log.Debugf("QuerySql entered")
	ok, t := common.CheckType(dbh, "*sqlx.DB")
	if !ok {
		err = fmt.Errorf("invalid dbh %s", t)
		return
	}
	log.Debugf("SQL: %s", mySQL)
	err = dbh.Select(result, mySQL, args...)
	return
}

// SelectSQL runs a query, add bind variables and returns a cursor
func SelectSQL(dbh *sqlx.DB, mySQL string, args ...any) (rows *sqlx.Rows, err error) {
	log.Debugf("QuerySql entered")
	ok, t := common.CheckType(dbh, "*sqlx.DB")
	if !ok {
		err = fmt.Errorf("invalid dbh %s", t)
		return
	}
	log.Debugf("SQL: %s", mySQL)
	rows, err = dbh.Queryx(mySQL, args...)
	return
}

// SelectStmt runs a query on a prepared statement, add bind variables and returns a cursor
func SelectStmt(stmt *sqlx.Stmt, args ...any) (rows *sqlx.Rows, err error) {
	log.Debugf("QueryStmt entered")
	ok, t := common.CheckType(stmt, "*sqlx.Stmt")
	if !ok {
		err = fmt.Errorf("invalid stmt %s", t)
		return
	}
	log.Debugf("Parameter values: %v", args...)
	rows, err = stmt.Queryx(args...)
	return
}

// MakeRowMap returns rows as a slice of map field->value
func MakeRowMap(rows *sqlx.Rows) (result []map[string]interface{}, err error) {
	log.Debugf("MakeRowMap entered")
	for rows.Next() {
		row := make(map[string]interface{})
		err = rows.MapScan(row)
		if err != nil {
			return
		}
		result = append(result, row)
	}
	return
}

// ExecSQL executes a sql on a open transaction and returns result handler
func ExecSQL(tx *sqlx.Tx, mySQL string, args ...any) (result sql.Result, err error) {
	log.Debugf("ExecSQL: %s", mySQL)
	ok, t := common.CheckType(tx, "*sqlx.Tx")
	if !ok {
		err = fmt.Errorf("invalid transaction %s", t)
		return
	}
	result, err = tx.Exec(mySQL, args...)
	return
}
