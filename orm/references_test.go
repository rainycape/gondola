package orm

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"gnd.la/orm/driver"
	"gnd.la/orm/driver/sql"
	"gnd.la/orm/driver/sqlite"
)

type BadEvent struct {
	Timestamp int `orm:",references=Timestamp"`
	Name      string
}

type Event struct {
	Id        int64 `orm:",primary_key,auto_increment"`
	Timestamp int64 `orm:",references=Timestamp"`
	Name      string
}

type EventProperty struct {
	Id      int64 `orm:",primary_key,auto_increment"`
	EventId int64 `orm:",references=Event"`
	Key     string
	Value   string
}

type TimedEvent struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Start int64 `orm:",references=Timestamp(Id)"` // This is the same that reference just Timestamp
	End   int64 `orm:",references=Timestamp"`
	Name  string
}

var (
	eventNames = []string{"E1", "E2", "E3"}
	eventCount = len(eventNames)
)

func testBadReferences(t *testing.T, o *Orm) {
	if o.Driver().Capabilities()&driver.CAP_JOIN == 0 {
		t.Log("skipping bad references test")
		return
	}
	// TODO: Test for bad references which omit the field
	// and bad references to non-existant field names
	_, err := o.Register((*BadEvent)(nil), &Options{
		Table: "test_references_bad_event",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Register((*Timestamp)(nil), &Options{
		Table: "test_references_bad_timestamp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Initialize(); err == nil || !strings.Contains(err.Error(), "type") {
		t.Errorf("expecting error when registering FK of different type, got %s instead", err)
	}
}

func testCount(t *testing.T, count int, expected int, msg string) {
	if expected < 0 {
		expected = eventCount
	}
	if count != expected {
		t.Errorf("expecting %d results for %s, got %d instead", expected, msg, count)
	}
}

func testEvent(t *testing.T, event *Event, pos int) {
	if pos < 0 {
		pos = len(eventNames) - pos
	}
	if expect := eventNames[pos]; expect != event.Name {
		t.Errorf("expecting event name %q, got %q instead", expect, event.Name)
	}
	if id := int64(pos + 1); id != event.Id {
		t.Errorf("expecting event id %d, got %d instead", id, event.Id)
	}
}

func testIterErr(t *testing.T, iter *Iter) {
	if err := iter.Err(); err != nil {
		t.Error(err)
	}
}

func testReferences(t *testing.T, o *Orm) {
	drv := o.Driver()
	if drv.Capabilities()&driver.CAP_JOIN == 0 {
		t.Log("skipping references test")
		return
	}
	if sdrv, ok := drv.(*sql.Driver); ok {
		_, isSQLite := sdrv.Backend().(*sqlite.Backend)
		if isSQLite {
			o.SqlDB().Exec("PRAGMA foreign_keys = ON")
		}
	}
	// Register Event first and then Timestamp. The ORM should
	// re-arrange them so Timestamp is created before Event.
	// TODO: Test for ambiguous joins, they don't work yet
	eventTable, err := o.Register((*Event)(nil), &Options{
		Table: "test_references_event",
	})
	if err != nil {
		t.Fatal(err)
	}
	timestampTable, err := o.Register((*Timestamp)(nil), &Options{
		Table: "test_references_timestamp",
		Name:  "Timestamp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Initialize(); err != nil {
		t.Fatal(err)
	}
	// Insert a few objects
	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)
	var timestamps []*Timestamp
	for _, v := range []time.Time{t1, t2} {
		ts := &Timestamp{Timestamp: v}
		if _, err := o.Insert(ts); err != nil {
			t.Fatal(err)
		}
		timestamps = append(timestamps, ts)
	}
	for _, v := range eventNames {
		if _, err := o.Insert(&Event{Timestamp: timestamps[0].Id, Name: v}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Insert(&Event{Name: "E4"}); err != nil {
		t.Fatal(err)
	}
	var timestamp *Timestamp
	var event *Event
	// Ambiguous query, should return an error
	iter := o.Query(Eq("Id", 1)).Iter()
	for iter.Next(&timestamp, &event) {
	}
	if err := iter.Err(); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expecting ambiguous query error, got %v instead", err)
	}
	var count int
	// Fetch all the events for timestamp with id=1
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Id", ASC).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1")
	testIterErr(t, iter)
	// Fetch all the events for timestamp with id=1, but ignore the timestamp
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Name", ASC).Iter()
	for count = 0; iter.Next((*Timestamp)(nil), &event); count++ {
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1")
	testIterErr(t, iter)
	// This should produce an untyped nil pointer error
	iter = o.Query(Eq("Timestamp|Id", 1)).Iter()
	for iter.Next(nil, &event) {
	}
	if err := iter.Err(); err != errUntypedNilPointer {
		t.Errorf("expecting error %s, got %s instead", errUntypedNilPointer, err)
	}
	// Fetch all the events for timestamp with id=1, but ignore the timestamp using an
	// explicit table.
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Name", ASC).Table(timestampTable.Skip().MustJoin(eventTable, nil, JoinTypeInner)).Iter()
	for count = 0; iter.Next((*Timestamp)(nil), &event); count++ {
		testEvent(t, event, count)
	}
	testIterErr(t, iter)
	testCount(t, count, -1, "timestamp Id=1")
	// Fetch all the events for timestamp with id=2. There are no events so event
	// should be nil.
	iter = o.Query(Eq("Timestamp|Id", 2)).Join(JoinTypeLeft).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if event != nil {
			t.Errorf("expecting nil event for Timestamp Id=2, got %+v instead", event)
		}
	}
	testCount(t, count, 1, "Timestamp Id=2")
	testIterErr(t, iter)
	// Fetch event with id=2 with its timestamp.
	iter = o.Query(Eq("Event|Id", 2)).Iter()
	for count = 0; iter.Next(&event, &timestamp); count++ {
		if event.Name != "E2" {
			t.Errorf("expecting event name E2, got %s instead", event.Name)
		}
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=2")
	testIterErr(t, iter)
	// Now do the same but pass (timestamp, event) to next. The ORM
	// should perform the join correctly anyway.
	iter = o.Query(Eq("Event|Id", 2)).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if event.Name != "E2" {
			t.Errorf("expecting event name E2, got %s instead", event.Name)
		}
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=2")
	testIterErr(t, iter)
	iter = o.Query(Eq("Event|Id", 4)).Table(eventTable.MustJoin(timestampTable, nil, JoinTypeLeft)).Iter()
	for count = 0; iter.Next(&event, &timestamp); count++ {
		if event.Name != "E4" {
			t.Errorf("expecting event name E4, got %s instead", event.Name)
		}
		if timestamp != nil {
			t.Errorf("expecting nil Timestamp, got %v instead", timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=4")
	testIterErr(t, iter)

	// Test reference spawned from a query
	iter = o.Query(Eq("Timestamp|Id", timestamps[0].Id)).Sort("Name", ASC).Iter()
	for count = 0; iter.Next(&event); count++ {
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1 with spawned reference")
	testIterErr(t, iter)
	// Violate the FK
	if _, err := o.Save(&Event{Timestamp: 1337}); err == nil {
		t.Error("expecting an error when violating FK")
	}
}

func test2LevelReferences(t *testing.T, o *Orm) {
	drv := o.Driver()
	if drv.Capabilities()&driver.CAP_JOIN == 0 {
		t.Log("skipping references test")
		return
	}
	if sdrv, ok := drv.(*sql.Driver); ok {
		_, isSQLite := sdrv.Backend().(*sqlite.Backend)
		if isSQLite {
			o.SqlDB().Exec("PRAGMA foreign_keys = ON")
		}
	}
	_, err := o.Register((*Event)(nil), &Options{
		Table: "test_2level_references_event",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Register((*Timestamp)(nil), &Options{
		Table: "test_2level_references_timestamp",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Register((*EventProperty)(nil), &Options{
		Table: "test_2level_references_event_property",
	})
	if err := o.Initialize(); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	ts := &Timestamp{
		// Remove subsecond precision, since the sqlite
		// backend can't handle subsecond precision. Also,
		// convert to UTC since the orm will always return UTC
		Timestamp: now.Add(-time.Nanosecond * time.Duration(now.Nanosecond())).UTC(),
	}
	if _, err := o.Insert(ts); err != nil {
		t.Fatal(err)
	}

	ev := &Event{
		Timestamp: ts.Id,
	}
	if _, err := o.Insert(ev); err != nil {
		t.Fatal(err)
	}

	evp := &EventProperty{
		EventId: ev.Id,
		Key:     "key",
		Value:   "value",
	}
	if _, err := o.Insert(evp); err != nil {
		t.Fatal(err)
	}
	var evp2 EventProperty
	q := Eq("Event|Timestamp|Id", ev.Timestamp)
	if _, err := o.One(q, &evp2); err != nil {
		t.Fatal(err)
	}
	compare := func(obj1, obj2 interface{}) {
		if !reflect.DeepEqual(obj1, obj2) {
			t.Errorf("expecting %T = %+v, got %+v instead", obj1, obj1, obj2)
		}
	}
	compare(evp, &evp2)
	// Try to retrieve other objects part of the join
	var ts2 Timestamp
	var ev2 Event
	if _, err := o.One(q, &evp2, &ts2); err != nil {
		t.Fatal(err)
	}
	compare(evp, &evp2)
	compare(ts, &ts2)
	if _, err := o.One(q, &ev2); err != nil {
		t.Fatal(err)
	}
	compare(ev, &ev2)
	if _, err := o.One(q, &ts2); err != nil {
		t.Fatal(err)
	}
	compare(ts, &ts2)
}
