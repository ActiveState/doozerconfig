package doozerconfig

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/doozer"
	"reflect"
)

type DoozerConfig struct {
	conn         *doozer.Conn
	configStruct interface{}
	prefix       string
	fields       map[string]reflect.Value
	fieldTypes   map[string]reflect.StructField
}

// New returns a new DoozerConfig given doozer connection, struct object and
// doozer path prefix.
func New(conn *doozer.Conn, configStruct interface{}, prefix string) *DoozerConfig {
	return &DoozerConfig{
		conn, configStruct, prefix,
		make(map[string]reflect.Value),
		make(map[string]reflect.StructField)}
}

// Load populates the struct with config values from doozer.
func (c *DoozerConfig) Load() error {
	elem := reflect.ValueOf(c.configStruct).Elem()
	elemType := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := elemType.Field(i)

		// read json-encoded bytes from doozer
		path := fieldType.Tag.Get("doozer")
		if path == "" {
			// this field is not supposed to be loaded from doozer because the
			// user did not provide a struct tag with doozer key for it.
			continue
		}
		path = c.prefix + path
		data, rev, err := c.conn.Get(path, nil)
		if err != nil {
			return err
		}
		// for some reason, Get returns rev=0 if the path doesn't exist
		if rev == 0 {
			return fmt.Errorf("doozerconfig: key %s does not exist in doozer", path)
		}

		c.fields[path] = field
		c.fieldTypes[path] = fieldType

		// decode the json and directly set the struct field		
		err = unmarshalIntoValue(data, field)
		if err != nil {
			return fmt.Errorf("doozerconfig: error decoding json from doozer[%s]: %v", path, err)
		}
	}
	return nil
}

// Monitor listens for config changes in doozer and updates the struct. A
// callback function can be passed to get notified of the changes made and errors.
func (c *DoozerConfig) Monitor(glob string, rev int64, callback func(string, interface{}, error)) {
	doozerWatch(c.conn, glob, rev, func(evt doozer.Event, err error) {
		if err != nil {
			callback("", nil, err)
			return // on doozer error, return immediately.
		}
		if field, ok := c.fields[evt.Path]; ok {
			err := unmarshalIntoValue(evt.Body, field)
			if err != nil {
				// on json errors, continue monitoring for more changes, but
				// report the error to the caller.
				callback("", nil, err)
			} else {
				callback(c.fieldTypes[evt.Path].Name, field.Interface(), nil)
			}
		}
	})
}

// a version of json.Unmarshal that unmarshalls into a reflect.Value type
func unmarshalIntoValue(data []byte, field reflect.Value) error {
	fieldInterface := field.Addr().Interface()
	return json.Unmarshal(data, &fieldInterface)
}

// monitor mutations on the given glob of keys and report them in the
// returned channel
func doozerWatch(c *doozer.Conn, glob string, rev int64, callback func(doozer.Event, error)) {
	for {
		evt, err := c.Wait(glob, rev)
		callback(evt, err)
		if err != nil {
			return
		}
		rev = evt.Rev + 1
	}
}
