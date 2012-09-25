// TODO: libraries should not use log.Fatal
// TODO: ... perhaps not log.Printf either.

package doozerconfig

import (
	"reflect"
	"github.com/ActiveState/doozer"
	"encoding/json"
	"errors" // TODO: create custom errors
	"log"
)

type DoozerConfig struct {
	conn        *doozer.Conn
	configStruct interface{}
	prefix       string
	fields       map[string]reflect.Value
}

func New(conn *doozer.Conn, configStruct interface{}, prefix string) *DoozerConfig{
	return &DoozerConfig{conn, configStruct, prefix, make(map[string]reflect.Value)}
}

// initialize the config data by loading from doozer.
// will return error if any of the config is not found
func (c *DoozerConfig) Load() error {
	elem := reflect.ValueOf(c.configStruct).Elem()
	elemType := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		fieldType := elemType.Field(i)

		// read json-encoded bytes from doozer
		path := fieldType.Tag.Get("doozer")
		if path == "" {
			// this field is not supposed to be loaded from doozer
			continue
		}
		path = c.prefix + path
		data, _, err := c.conn.Get(path, nil)
		if err != nil {
			return err
		}

		c.fields[path] = field
		
		// extract the value based on the field type
		err = setFieldWithData(field, data)
		if err != nil {
			return err
		}
	}
	return nil
}


func (c *DoozerConfig) Monitor(glob string, rev int64) {
	for evt := range doozerWatch(c.conn, glob, rev) {
		if field, ok := c.fields[evt.Path]; ok {
			err := setFieldWithData(field, evt.Body)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("New Config: %+v\n", c.configStruct)
		}
	}
}


// set this struct field with data json-decoded as the same tyhpe
func setFieldWithData(field reflect.Value, data []byte) error {
	// TODO: simplify this using interface{}
	switch(field.Kind()){
	case reflect.Int:
		var val int64
		json.Unmarshal(data, &val)
		field.SetInt(val)
	case reflect.String:
		var val string
		json.Unmarshal(data, &val)
		field.SetString(val)
	default:
		return errors.New("doozerconfig: unsupported field " + string(field.Kind()))
	}
	return nil
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
				// FIXME: on doozer watch errors, the entire basin process
				// must not go down. figure a way to report the error in
				// console and silently proceed.
				log.Fatal(err)
				return
			}
			rev = evt.Rev + 1
			ch <- evt
		}
	}()
	return ch
}

