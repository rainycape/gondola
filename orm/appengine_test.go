// +build appengine

package orm

import (
	"testing"

	"gnd.la/orm/driver/datastore"

	"appengine/aetest"
)

func TestDatastore(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	o := newOrm(t, "datastore://", true)
	o.Driver().(*datastore.Driver).SetContext(c)
	testOrm(t, o)
}
