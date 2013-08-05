package orm

import (
	"fmt"
	_ "gondola/orm/drivers/postgres"
	_ "gondola/orm/drivers/sqlite"
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
	o := newOrm(t, "postgres", fmt.Sprintf("dbname=gotest user=%v password=%v", u.Username, u.Username), true)
	testOrm(t, o)
	o.Close()
}
