package test

import (
	"database/sql"
	"os"
	"testing"

	ora "github.com/sijms/go-ora/v2"

	"github.com/tommi2day/gomodules/dblib"

	_ "github.com/glebarez/go-sqlite"
	"github.com/sijms/go-ora/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var urlOptions = map[string]string{
	// "CONNECTION TIMEOUT": "3",
}

func makeOerr(code int, msg string) *network.OracleError {
	e := &network.OracleError{
		ErrCode: code,
		ErrMsg:  msg,
	}
	return e
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
		db, err := dblib.DBConnect("sqlite", ":memory:", 5)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)
		require.NoError(t, err, "DB Open sqlite memory failed")
		assert.NotEmpty(t, db, "DB Handle missed")
	})

	t.Run("Test DB Connect noexisting oracle", func(t *testing.T) {
		service := "xxx"
		connect := ora.BuildJDBC("dummy", "dummy", service, urlOptions)
		_, err := dblib.DBConnect("oracle", connect, 5)
		assert.Error(t, err, "DB Open oracle should fail")
	})

	t.Run("Test DB Connect existing file", func(t *testing.T) {
		filename := "test2.db"
		err := newTestfile(filename)
		if err != nil {
			t.Fatalf("Cannot create sqlite file")
		}
		db, err := dblib.DBConnect("sqlite", filename, 5)
		require.NoError(t, err, "DB Open sqlite %s failed")
		assert.NotEmpty(t, db, "DB Handle missed")
		e := db.Close()
		if e == nil {
			_ = os.Remove(filename)
		}
	})
}

func TestSelectOneStringValue(t *testing.T) {
	var actual string
	var err error
	filename := "test3.db"
	err = newTestfile(filename)
	if err != nil {
		t.Fatalf("Cannot create sqlite file")
	}
	db, err := dblib.DBConnect("sqlite", filename, 5)
	defer func(db *sql.DB) {
		e := db.Close()
		if e == nil {
			_ = os.Remove(filename)
		}
	}(db)
	if err != nil {
		t.Fatalf("DB Open sqlite %s failed", filename)
	}
	if db == nil {
		t.Fatalf("DB Handle missed")
	}

	t.Run("Test Select Singlerow", func(t *testing.T) {
		mysql := "select sqlite_version()"
		actual, err = dblib.SelectOneStringValue(db, mysql)
		assert.NoErrorf(t, err, "Querry returned error %s", err)
		assert.NotEmpty(t, actual, "Select value empty")
		t.Logf("Version %s", actual)
	})
	t.Run("Test wrong sql", func(t *testing.T) {
		mysql := "seleccct sqlite_version()"
		_, err = dblib.SelectOneStringValue(db, mysql)
		assert.Error(t, err, "Query returned no error, but should")
	})
}
