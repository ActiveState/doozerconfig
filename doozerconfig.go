package doozerconfig

import (
	"reflect"
	"github.com/ActiveState/doozer"
	"encoding/json"
	"errors" // TODO: create custom errors
)

type DoozerConfig struct {
	conn        *doozer.Conn
	configStruct interface{}
	prefix       string
}

func New(conn *doozer.Conn, configStruct interface{}, prefix string) *DoozerConfig{
	return &DoozerConfig{conn, configStruct, prefix}
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
		data, _, err := c.conn.Get(c.prefix + path, nil)
		if err != nil {
			return err
		}

		// decode the json into interface{} type
		var val2 interface{}
		json.Unmarshal(data, &val2)
		
		// extract the value based on the field type
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
	}
	return nil
}

