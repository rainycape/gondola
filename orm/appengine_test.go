// +build appengine

package orm

import (
	"testing"

	"gnd.la/orm/driver/datastore"

	"appengine/aetest"
)

type datastoreOpener struct {
}

func (o *datastoreOpener) Open(t T) (*Orm, interface{}) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	orm := newOrm(t, "datastore://", true)
	orm.Driver().(*datastore.Driver).SetContext(c)
	return orm, c
}

func (o *datastoreOpener) Close(data interface{}) {
	data.(aetest.Context).Close()
}

func TestDatastore(t *testing.T) {
	runAllTests(t, &datastoreOpener{})
}

func init() {
	openers["default"] = &datastoreOpener{}
	openers["datastore"] = &datastoreOpener{}
}
