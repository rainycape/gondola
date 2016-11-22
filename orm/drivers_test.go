// +build !appengine

package orm

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"testing"

	"net"

	"path/filepath"

	_ "gnd.la/orm/driver/mysql"
	_ "gnd.la/orm/driver/postgres"
	_ "gnd.la/orm/driver/sqlite"
)

// This file has tests which run all the tests for
// every driver.

type sqliteOpener struct {
}

func (o *sqliteOpener) Open(t testing.TB) (*Orm, interface{}) {
	f, err := ioutil.TempFile("", "sqlite-")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	orm := newOrm(t, "sqlite://"+f.Name(), true)
	orm.SqlDB().Exec("PRAGMA journal_mode = WAL")
	orm.SqlDB().Exec("PRAGMA foreign_keys = on")
	return orm, f.Name()
}

func (o *sqliteOpener) Close(data interface{}) {
	// sqlite will create multiple files per db,
	// since we're using journal_mode = WAL
	files, err := filepath.Glob(data.(string) + "*")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		os.Remove(f)
	}
}

func (o *sqliteOpener) Name() string { return "sqlite" }

type postgresOpener struct {
}

func (o *postgresOpener) Open(t testing.TB) (*Orm, interface{}) {
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	exec.Command("dropdb", "gotest").Run()
	if err := exec.Command("createdb", "gotest").Run(); err != nil {
		t.Skip("cannot create gotest postgres database, skipping test")
	}
	return newOrm(t, fmt.Sprintf("postgres://dbname=gotest user=%v password=%v", u.Username, u.Username), true), nil
}

func (o *postgresOpener) Close(_ interface{}) {}
func (o *postgresOpener) Name() string        { return "postgresql" }

type mysqlOpener struct {
}

func (o *mysqlOpener) Open(t testing.TB) (*Orm, interface{}) {
	// Check if MySQL is running
	conn, err := net.Dial("tcp", "localhost:3306")
	if err != nil {
		t.Skipf("MySQL is not running, skipping test (%v)", err)
	}
	conn.Close()
	creds := os.Getenv("GONDOLA_ORM_MYSQL_CREDENTIALS")
	if creds == "" {
		creds = "gotest:gotest"
	}
	orm := newOrm(t, "mysql://"+creds+"@/", true)
	db := orm.SqlDB()
	if _, err := db.Exec("DROP DATABASE IF EXISTS gotest"); err != nil {
		t.Skipf("cannot connect to mysql database, skipping test: %s", err)
	}
	if _, err := db.Exec("CREATE DATABASE gotest"); err != nil {
		t.Fatal(err)
	}
	if err := orm.Close(); err != nil {
		t.Fatal(err)
	}
	return newOrm(t, "mysql://"+creds+"@/gotest", true), nil
}

func (o *mysqlOpener) Close(_ interface{}) {}
func (o *mysqlOpener) Name() string        { return "mysql" }

func TestSqlite(t *testing.T) {
	runAllTests(t, &sqliteOpener{})
}

func TestPostgres(t *testing.T) {
	runAllTests(t, &postgresOpener{})
}

func TestMysql(t *testing.T) {
	runAllTests(t, &mysqlOpener{})
}

func init() {
	openers["default"] = &sqliteOpener{}
	openers["sqlite"] = &sqliteOpener{}
	openers["postgres"] = &postgresOpener{}
	openers["mysql"] = &mysqlOpener{}
}
