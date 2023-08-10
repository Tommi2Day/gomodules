package dblib

import (
	"database/sql"
	"os"
	"testing"

	"github.com/tommi2day/gomodules/common"

	"github.com/jmoiron/sqlx"

	ora "github.com/sijms/go-ora/v2"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var urlOptions = map[string]string{
	// "CONNECTION TIMEOUT": "3",
}

func newTestfile(filename string) error {
	var err error
	var f *os.File
	//nolint gosec
	f, err = os.Create(filename)
	if err != nil {
		return err
	}
	err = f.Close()
	return err
}

func TestDBConnect(t *testing.T) {
	t.Run("Test DB Connect Memory", func(t *testing.T) {
		dbh, err := DBConnect("sqlite", ":memory:", 5)
		defer func(dbh *sqlx.DB) {
			_ = dbh.Close()
		}(dbh)
		require.NoError(t, err, "DB Open sqlite memory failed")
		assert.NotEmpty(t, dbh, "DB Handle missed")
	})

	t.Run("Test DB Connect noexisting oracle", func(t *testing.T) {
		service := "xxx"
		connect := ora.BuildJDBC("dummy", "dummy", service, urlOptions)
		_, err := DBConnect("oracle", connect, 5)
		assert.Error(t, err, "DB Open oracle should fail")
	})

	t.Run("Test DB Connect existing file", func(t *testing.T) {
		filename := "test2.db"
		err := newTestfile(filename)
		if err != nil {
			t.Fatalf("Cannot create sqlite file")
		}
		dbh, err := DBConnect("sqlite", filename, 5)
		require.NoError(t, err, "DB Open sqlite %s failed")
		assert.NotEmpty(t, dbh, "DB Handle missed")
		e := dbh.Close()
		if e == nil {
			_ = os.Remove(filename)
		}
	})
}
func TestSQL(t *testing.T) {
	var actualString string
	var actualInt int64
	var err error
	var actualRows int64
	var rows *sqlx.Rows
	var res sql.Result
	var dbh *sqlx.DB
	var stmt *sqlx.Stmt
	type testTab struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	filename := "test3.db"
	err = newTestfile(filename)
	if err != nil {
		t.Fatalf("Cannot create sqlite file")
	}
	dbh, err = DBConnect("sqlite", filename, 5)
	defer func(dbh *sqlx.DB) {
		e := dbh.Close()
		if e == nil {
			_ = os.Remove(filename)
		}
	}(dbh)
	if err != nil {
		t.Fatalf("DB Open sqlite %s failed", filename)
	}
	if dbh == nil {
		t.Fatalf("DB Handle missed")
	}
	t.Run("Test DoSql Create", func(t *testing.T) {
		mysql := "create table test (id int, name varchar(20))"
		res, err = ExecSQL(dbh, mysql)
		assert.NoErrorf(t, err, "Query returned error %s", err)
	})
	t.Run("Test DoSql insert", func(t *testing.T) {
		mysql := "insert into test (id, name) values (1, 'test')"
		res, err = ExecSQL(dbh, mysql)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		require.NotNil(t, res, "Result is nil")
		actualRows, err = res.RowsAffected()
		assert.Equal(t, int64(1), actualRows, "Rows not as expected")
	})
	t.Run("Test DoSql insert parameter", func(t *testing.T) {
		mysql := "insert into test (id, name) values (2, 'test2')"
		stmt, err = PrepareSQL(dbh, mysql)
		require.NoError(t, err, "Prepare returned error %s", err)
		res, err = ExecStmt(dbh, stmt, 2, "test2")
		assert.NoErrorf(t, err, "Query returned error %s", err)
		require.NotNil(t, res, "Result is nil")
		actualRows, err = res.RowsAffected()
		assert.Equal(t, int64(1), actualRows, "Rows not as expected")
		t.Logf("Value %d", actualRows)
	})

	t.Run("Test wrong sql", func(t *testing.T) {
		mysql := "seleccct sqlite_version()"
		_, err = SelectOneStringValue(dbh, mysql)
		assert.Error(t, err, "Query returned no error, but should")
	})
	t.Run("Test Select Singlerow String", func(t *testing.T) {
		mysql := "select name from test where id = ?"
		actualString, err = SelectOneStringValue(dbh, mysql, 1)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.NotEmpty(t, actualString, "Select value empty")
		assert.Equal(t, "test", actualString, "Select value not as expected")
		t.Logf("Version %s", actualString)
	})

	t.Run("Test Select Singlerow Int64", func(t *testing.T) {
		mysql := "select id from test where name = 'test'"
		actualInt, err = SelectOneInt64Value(dbh, mysql)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.Equal(t, int64(1), actualInt, "Select value empty")
		t.Logf("Value %d", actualInt)
	})
	t.Run("Test wrong result type", func(t *testing.T) {
		mysql := "select name from test where id = 1"
		_, err = SelectOneInt64Value(dbh, mysql)
		assert.Error(t, err, "Query returned no error, but should")
	})
	t.Run("test wrong dbh", func(t *testing.T) {
		mysql := "select sqlite_version()"
		_, err = SelectOneInt64Value(nil, mysql)
		assert.Error(t, err, "Query returned no error, but should")
	})
	t.Run("test CheckType", func(t *testing.T) {
		actual, e := common.CheckType(dbh, "test")
		assert.False(t, actual, "type returned true, but should not")
		assert.NotEmpty(t, e, "actual is empty, but should not")
	})
	t.Run("Test query one row", func(t *testing.T) {
		var row testTab
		mysql := "select id, name from test where id = ?"
		err = SelectOneRow(dbh, mysql, &row, 1)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.Equal(t, int64(1), row.ID, "id value not expected")
		assert.Equal(t, "test", row.Name, "name value not expected")
	})
	t.Run("Test query all rows", func(t *testing.T) {
		var result []testTab
		mysql := "select id,name from test"
		err = SelectAllRows(dbh, mysql, &result)
		c := len(result)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.Equal(t, 2, c, "number of rows not expected")
		if c > 1 {
			row := result[1]
			assert.Equal(t, int64(2), row.ID, "id value not expected")
			assert.Equal(t, "test2", row.Name, "name value not expected")
		}
	})

	tx := dbh.MustBegin()
	t.Run("Test ExecSQLTx", func(t *testing.T) {
		mysql := "insert into test (id, name) values (?, ?)"
		res, err = ExecSQLTx(tx, mysql, 3, "test3")
		assert.NoErrorf(t, err, "Query returned error %s", err)
		require.NotNil(t, res, "Result is nil")
		actualRows, err = res.RowsAffected()
		assert.Equal(t, int64(1), actualRows, "Rows not as expected")
	})
	_ = tx.Commit()
	tx = dbh.MustBegin()
	t.Run("Test prepared named statement with transaction ", func(t *testing.T) {
		mysql := "insert into test (id, name) values ($1, $2)"
		stmt, err = PrepareSQLTx(tx, mysql)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.NotNil(t, stmt, "Statement is nil")
	})
	t.Run("Test ExecStmtTx", func(t *testing.T) {
		require.NotNil(t, stmt)
		res, err = ExecStmtTx(tx, stmt, 4, "rollback")
		assert.NoErrorf(t, err, "Query returned error %s", err)
		require.NotNil(t, res, "Result is nil")
		actualRows, err = res.RowsAffected()
		assert.Equal(t, int64(1), actualRows, "Rows not as expected")
	})
	_ = tx.Rollback()
	t.Run("Test Rollback", func(t *testing.T) {
		mysql := "select id from test where name = 'rollback'"
		actualInt, err = SelectOneInt64Value(dbh, mysql)
		assert.Error(t, err, "Query should return error")
		t.Logf("Value %d", actualInt)
	})

	t.Run("Test prepare without transaction", func(t *testing.T) {
		mysql := "select * from test where id > ?"
		stmt, err = PrepareSQL(dbh, mysql)
		assert.NoErrorf(t, err, "Query returned error %s", err)
		assert.NotNil(t, stmt, "Statement is nil")
		if stmt != nil {
			err = stmt.QueryRow(1).Scan(&actualInt, &actualString)
			assert.NoErrorf(t, err, "Query returned error %s", err)
			assert.Equal(t, int64(2), actualInt, "id value not expected")
			assert.Equal(t, "test2", actualString, "name value not expected")
		}
	})
	t.Run("Test Select SQL with stmt return struct", func(t *testing.T) {
		rows, err = SelectStmt(stmt, 0)
		assert.NoErrorf(t, err, "Query should not return error:%s", err)
		assert.NotNil(t, rows, "Rows is nil")
		var result []testTab
		for rows.Next() {
			var record testTab
			err = rows.StructScan(&record)
			assert.NoErrorf(t, err, "Query returned error %s", err)
			result = append(result, record)
		}
		assert.Equal(t, 3, len(result), "number of rows not expected")
		t.Logf("Rows %v", result)
		_ = rows.Close()
	})
	t.Run("Test Select SQL returning map", func(t *testing.T) {
		mysql := "select * from test"
		rows, err = SelectSQL(dbh, mysql)
		assert.NoErrorf(t, err, "Query should not return error:%s", err)
		assert.NotNil(t, rows, "Rows is nil")
		var result []map[string]interface{}
		result, err = MakeRowMap(rows)
		assert.NoErrorf(t, err, "Mapping returned error %s", err)
		assert.Equal(t, 3, len(result), "number of rows not expected")
		t.Logf("Rows %v", result)
		_ = rows.Close()
	})
	t.Run("Test Select SQL with stmt and no rows", func(t *testing.T) {
		rows, err = SelectStmt(stmt, 3)
		assert.NoErrorf(t, err, "Query should not return error:%s", err)
		assert.NotNil(t, rows, "Rows is nil")
		var result []testTab
		for rows.Next() {
			var record testTab
			err = rows.StructScan(&record)
			assert.NoErrorf(t, err, "Query returned error %s", err)
			result = append(result, record)
		}
		assert.Equal(t, 0, len(result), "number of rows not expected")
		t.Logf("Rows %v", result)
		_ = rows.Close()
	})
	t.Run("Test Select with wrong sql", func(t *testing.T) {
		mysql := "select * from test where idxx > ?"
		rows, err = SelectSQL(dbh, mysql)
		assert.Error(t, err, "Query should return error")
		assert.Nil(t, rows, "Rows is not nil")
	})
}
