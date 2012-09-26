package doozerconfig

import (
	"reflect"
	"github.com/ActiveState/doozer"
	"encoding/json"
	"log"
	"fmt"
)

type DoozerConfig struct {
	conn        *doozer.Conn
	configStruct interface{}
	prefix       string
	fields       map[string]reflect.Value
	fieldTypes   map[string]reflect.StructField
}

// New returns a new DoozerConfig given doozer connection, struct object and
// doozer path prefix.
func New(conn *doozer.Conn, configStruct interface{}, prefix string) *DoozerConfig{
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

// pair of field and value, of a struct field
type FieldPair struct {
	Value reflect.Value
	Field reflect.StructField
}

// Monitor monitors new mutations in the given path glob and, if they are
// config keys, updates the struct fields accordingly.
func (c *DoozerConfig) Monitor(glob string, rev int64) chan FieldPair {
	ch := make(chan FieldPair)
	go func(){
		for evt := range doozerWatch(c.conn, glob, rev) {
			if field, ok := c.fields[evt.Path]; ok {
				err := unmarshalIntoValue(evt.Body, field)
				if err != nil {
					log.Fatal(err)
				}
				ch <- FieldPair{field, c.fieldTypes[evt.Path]}
			}
		}
	}()
	return ch
}


// a version of json.Unmarshal that unmarshalls into a reflect.Value type
func unmarshalIntoValue(data []byte, field reflect.Value) error {
	fieldInterface := field.Addr().Interface()
	return json.Unmarshal(data, &fieldInterface)
}


// monitor mutations on the given glob of keys and report them in the
// returned channel
func doozerWatch(c *doozer.Conn, glob string, rev int64) chan doozer.Event {
	ch := make(chan doozer.Event)
	go func() {
		for {
			evt, err := c.Wait(glob, rev)
			if err != nil {
				close(ch)
				// FIXME: use error channels; a library should not crash the
				// program.
				log.Fatal(err)
				return
			}
			rev = evt.Rev + 1
			ch <- evt
		}
	}()
	return ch
}

