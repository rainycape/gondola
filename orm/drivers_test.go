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

func testOrm(t *testing.T, o *Orm) {
	tests := []func(*testing.T, *Orm){
		testCodecs,
		testAutoIncrement,
		testTime,
		testSaveDelete,
		testLoadSaveMethods,
		testLoadSaveMethodsErrors,
		testData,
		testInnerPointer,
		testTransactions,
		testCompositePrimaryKey,
		testReferences,
		testQueryAll,
	}
	for _, v := range tests {
		v(t, o)
	}
}

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
	exec.Command("createdb", "gotest").Run()
	o := newOrm(t, fmt.Sprintf("postgres://dbname=gotest user=%v password=%v", u.Username, u.Username), true)
	testOrm(t, o)
	o.Close()
}

func TestMysql(t *testing.T) {
	o := newOrm(t, "mysql://gotest:gotest@/gotest", true)
	db := o.SqlDB()
	db.Exec("DROP DATABASE IF EXISTS gotest")
	db.Exec("CREATE DATABASE gotest")
	db.Exec("USE gotest")
	testOrm(t, o)
	o.Close()
}
