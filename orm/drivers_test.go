// +build !appengine

package orm

import (
	"fmt"
	_ "gnd.la/orm/driver/mysql"
	_ "gnd.la/orm/driver/postgres"
	_ "gnd.la/orm/driver/sqlite"
	"os"
	"os/exec"
	"os/user"
	"testing"
)

// This file has tests which run all the tests for
// every driver.

func TestSqlite(t *testing.T) {
	name, o := newTmpOrm(t)
	defer o.Close()
	defer os.Remove(name)
	testOrm(t, o)
}

func TestPostgres(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	exec.Command("dropdb", "gotest").Run()
	if err := exec.Command("createdb", "gotest").Run(); err != nil {
		t.Skip("cannot create gotest postgres database, skipping test")
	}
	o := newOrm(t, fmt.Sprintf("postgres://dbname=gotest user=%v password=%v", u.Username, u.Username), true)
	testOrm(t, o)
	o.Close()
}

func TestMysql(t *testing.T) {
	o := newOrm(t, "mysql://gotest:gotest@/test", true)
	db := o.SqlDB()
	if _, err := db.Exec("DROP DATABASE IF EXISTS gotest"); err != nil {
		t.Skipf("cannot connect to mysql database, skipping test: %s", err)
	}
	if _, err := db.Exec("CREATE DATABASE gotest"); err != nil {
		t.Fatal(err)
	}
	if err := o.Close(); err != nil {
		t.Fatal(err)
	}
	o = newOrm(t, "mysql://gotest:gotest@/gotest", true)
	testOrm(t, o)
	o.Close()
}
