package driver

import (
	"reflect"
	"testing"
	"time"
)

func TestZero(t *testing.T) {
	values := []interface{}{
		0,
		0.0,
		time.Time{},
		false,
		nil,
	}
	for _, v := range values {
		if !IsZero(reflect.ValueOf(v)) {
			t.Errorf("zero not detected for %v (%T)", v, v)
		}
	}
}
