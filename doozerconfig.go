package doozerconfig

import (
	"encoding/json"
	"fmt"
	"github.com/ActiveState/doozer"
	"path/filepath"
	"reflect"
	"strings"
)

type DoozerConfig struct {
	conn         *doozer.Conn
	loadRev      int64
	configStruct interface{}
	prefix       string
	fields       map[string]reflect.Value
	fieldTypes   map[string]reflect.StructField
}

// New returns a new DoozerConfig given doozer connection, struct object and
// doozer path prefix.
func New(conn *doozer.Conn, loadRev int64, configStruct interface{}, prefix string) *DoozerConfig {
	return &DoozerConfig{
		conn, loadRev, configStruct, prefix,
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

		path := fieldType.Tag.Get("doozer")
		if path == "" {
			continue // not a field to be loaded from doozer
		}
		path = c.prefix + path

		if fieldType.Type.Kind() == reflect.Map {
			keyKind := fieldType.Type.Key().Kind()

			if keyKind != reflect.String {
				panic("map field must have string key type")
			}

			list, err := c.conn.Getdirinfo(path, c.loadRev, 0, -1)
			if err != nil {
				return fmt.Errorf("doozerconfig: error listing '%s': %s", path, err)
			}
			for _, fileinfo := range list {
				if fileinfo.IsDir {
					fmt.Printf("doozerconfig warning: ignoring child that is a dir - %s", fileinfo.Name)
					continue
				}
				data, _, err := c.conn.Get(path+"/"+fileinfo.Name, &c.loadRev)
				if err != nil {
					return err
				}
				err = setJsonValueOnMap(field, fileinfo.Name, data)
				if err != nil {
					return err
				}
			}
		} else {
			data, rev, err := c.conn.Get(path, nil)
			if err != nil {
				return fmt.Errorf("doozerconfig: error reading '%s' from doozer: %s", path, err)
			}
			// Get returns rev=0 if the path doesn't exist
			if rev == 0 {
				return fmt.Errorf("doozerconfig: key %s does not exist in doozer", path)
			}

			// decode the json and directly set the struct field		
			err = setJsonValue(field, data)
			if err != nil {
				return fmt.Errorf("doozerconfig: error decoding json from doozer[%s]: %v", path, err)
			}
		}
		c.fields[path] = field
		c.fieldTypes[path] = fieldType
	}
	return nil
}

// A Change type represents the kind of change that happened
type ChangeType uint

const (
	SET ChangeType = iota
	DELETE
)

// Change represents a change to the config struct
type Change struct {
	FieldName string // Field name that was changed
	Type      ChangeType
	Key       string      // If field is map, key value that changed
	Value     interface{} // New value; if map, value is slotted at changed key
}

// Monitor listens for config changes in doozer and updates the struct. A
// callback function can be passed to get notified of the changes made and errors.
func (c *DoozerConfig) Monitor(glob string, callback func(*Change, error)) {
	if callback == nil {
		panic("Monitor requires a non-nil callback argument")
	}
	doozerWatch(c.conn, glob, c.loadRev, func(evt doozer.Event, err error) {
		if err != nil {
			callback(nil, err)
			return // on doozer error, return immediately.
		}
		change, err := c.handleMutation(evt)
		// TODO: use oldValue
		if err != nil {
			// on json errors, continue monitoring for more changes, but
			// report the error to the caller.
			callback(change, err)
		} else if change != nil {
			callback(change, nil)
		}
	})
}

func (c *DoozerConfig) handleMutation(evt doozer.Event) (*Change, error) {
	if field, ok := c.fields[evt.Path]; ok && evt.IsSet() {
		// Mutation of simple types
		name := c.fieldTypes[evt.Path].Name
		err := setJsonValue(field, evt.Body)
		if err != nil {
			return nil, err
		} else {
			return &Change{name, SET, "", field.Interface()}, nil
		}
	} else {
		parent, name := filepath.Split(evt.Path)
		if parent != "" {
			parent = strings.TrimRight(parent, "/")
			if field, ok := c.fields[parent]; ok {
				// Mutation of map type
				if evt.IsSet() {
					err := setJsonValueOnMap(field, name, evt.Body)
					if err != nil {
						return nil, err
					}
					return &Change{name, SET, name, field.Interface()}, nil
				}
				if evt.IsDel() {
					err := delMapKey(field, name)
					if err != nil {
						return nil, err
					}
					return &Change{name, DELETE, name, nil}, nil
				}
			}
		}
	}
	// ignore; unknown path
	return nil, nil
}

// setJsonValue sets `field` to contain the json-decoded value
func setJsonValue(field reflect.Value, data []byte) error {
	fieldInterface := field.Addr().Interface()
	return json.Unmarshal(data, &fieldInterface)
}

// setJsonValueOnMap sets dict[key] to `data` json-decoded to the same type.
// dict must refer to non-nil map, else panic.
func setJsonValueOnMap(dict reflect.Value, key string, data []byte) error {
	elemType := dict.Type().Elem()
	switch elemType.Kind() {
	case reflect.String, reflect.Int:
		// OK
	default:
		return fmt.Errorf("unsupported map value type")
	}
	value := reflect.New(elemType)
	err := json.Unmarshal(data, value.Interface())
	if err != nil {
		return err
	}
	dict.SetMapIndex(reflect.ValueOf(key), value.Elem())
	return nil
}

func delMapKey(dict reflect.Value, key string) error {
	dict.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
	return nil
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
