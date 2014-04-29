package orm

import (
	"testing"
)

type Referenced struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Value string
}

type Migration1 struct {
	Id int64 `orm:",primary_key,auto_increment"`
}

type BadMigration1 struct {
	Id    int64  `orm:",primary_key,auto_increment"`
	Value string `orm:",notnull"`
}

type Migration2 struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Value string
}

type BadMigration2 struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Value int
}

type Migration3 struct {
	Id     int64  `orm:",primary_key,auto_increment"`
	Value2 string `orm:",notnull,default=Gondola"`
}

type Migration4 struct {
	Id        int64 `orm:",primary_key,auto_increment"`
	Reference int64 `orm:",references=Referenced"`
}

var (
	migrationOptions = &Options{Name: "Migration", Table: "migration"} // This ensures the same table is always used
)

func testMigrations(t *testing.T, o *Orm) {
	clearRegistry := func() {
		globalRegistry.names = make(map[string]nameRegistry)
	}
	o.mustRegister((*Migration1)(nil), migrationOptions)
	o.mustInitialize()
	clearRegistry()
	o.mustRegister((*BadMigration1)(nil), migrationOptions)
	if err := o.Initialize(); err == nil {
		t.Error("expecting an error when initializing BadMigration1")
	} else {
		t.Logf("got expected error: %s", err)
	}
	clearRegistry()
	o.mustRegister((*Migration2)(nil), migrationOptions)
	if err := o.Initialize(); err != nil {
		t.Errorf("error initizing Migration2: %s", err)
	}
	if _, err := o.Insert(&Migration2{Value: "Gondola"}); err != nil {
		t.Errorf("error inserting Migration2: %s", err)
	}
	clearRegistry()
	o.mustRegister((*BadMigration2)(nil), migrationOptions)
	if err := o.Initialize(); err == nil {
		t.Error("expecting an error when initializing BadMigration2")
	} else {
		t.Logf("got expected error: %s", err)
	}
	clearRegistry()
	o.mustRegister((*Migration3)(nil), migrationOptions)
	if err := o.Initialize(); err != nil {
		t.Errorf("error initizing Migration3: %s", err)
		panic(err)
	}
	m3 := &Migration3{}
	if _, err := o.Insert(m3); err != nil {
		t.Errorf("error inserting Migration3: %s", err)
	}
	// We should get a different Migration3, generated from the
	// previous Migration2 INSERT. Its value should match the
	// default.
	var m3c *Migration3
	if found, err := o.All().Sort("Id", ASC).One(&m3c); err != nil {
		t.Errorf("error querying Migration3: %s", err)
	} else if !found {
		t.Error("m3 not found")
	} else {
		if m3c.Id == m3.Id {
			t.Errorf("expecting different Id for m3c, got same %v", m3c.Id)
		}
		if m3c.Value2 != "Gondola" {
			t.Errorf("expecting m3.Value2 = \"Gondola\", got %q instead", m3c.Value2)
		}
	}
	clearRegistry()
	o.mustRegister((*Referenced)(nil), nil)
	o.mustRegister((*Migration4)(nil), migrationOptions)
	if err := o.Initialize(); err != nil {
		t.Errorf("error initializing Migration4: %s", err)
	}
	if _, err := o.Insert(&Migration4{Reference: 42}); err == nil {
		t.Error("expecting an error when violating Migration4 FK")
	}
	ref := &Referenced{}
	o.MustInsert(ref)
	if _, err := o.Insert(&Migration4{Reference: ref.Id}); err != nil {
		t.Error(err)
	}
}

func TestMigrations(t *testing.T) {
	runTest(t, testMigrations)
}

/*func TestBadMigration1(t *testing.T) {
	runTest(t, testBadMigration1)
}*/
