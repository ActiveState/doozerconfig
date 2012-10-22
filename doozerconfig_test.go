// before running the test, install and run doozerd,
// https://github.com/ActiveState/doozerd/wiki/Instructions-to-compile-doozer

package doozerconfig

import (
	"fmt"
	"github.com/ActiveState/doozer"
	"testing"
	"time"
)

func TestSimple(_t *testing.T) {
	t := NewDoozerConfigTest("simple", _t)
	defer t.Close()
	t.DoozerSet("/prefix/foo", "42")
	t.DoozerSet("/prefix/bar", "\"hello world\"")
	var Config struct {
		Foo int    `doozer:"foo"`
		Bar string `doozer:"bar"`
	}
	t.Load(&Config, "/prefix/")
	if Config.Foo != 42 {
		t.Fatal("Foo is not 42")
	}
	if Config.Bar != "hello world" {
		t.Fatal("Bar is not 'hello world'")
	}
}

func TestMonitor(_t *testing.T) {
	t := NewDoozerConfigTest("monitor", _t)
	defer t.Close()
	t.DoozerSet("/foo", "42")
	var Config struct {
		Foo int `doozer:"/foo"`
	}
	t.Load(&Config, "")
	go t.Monitor("/foo*")
	t.DoozerSet("/foo", "69")
	time.Sleep(100 * time.Millisecond)
	if Config.Foo != 69 {
		t.Fatalf("Config.Foo was not modified; value=%s", Config.Foo)
	}
}

func TestMapType(_t *testing.T) {
	t := NewDoozerConfigTest("maptype", _t)
	defer t.Close()
	t.DoozerSet("/dict/foo", `"hello"`)
	t.DoozerSet("/dict/bar", `"world"`)
	var Config struct {
		Dict map[string]string `doozer:"/dict"`
	}
	Config.Dict = make(map[string]string)
	t.Load(&Config, "")
	if Config.Dict["foo"] != "hello" {
		t.Fatalf("Config.Foo is not set")
	}
	if Config.Dict["bar"] != "world" {
		t.Fatalf("Config.Bar is not set")
	}
	// test mutation on map fields
	go t.Monitor("/dict/*")

	// .. when a key is changed:
	t.DoozerSet("/dict/bar", `"you"`)
	time.Sleep(100 * time.Millisecond)
	if Config.Dict["bar"] != "you" {
		t.Fatalf("Config.Bar did not change after mutation; value=%s", Config.Dict["bar"])
	}

	// .. when a key is added:
	t.DoozerSet("/dict/new", `"hello again"`)
	time.Sleep(100 * time.Millisecond)
	if Config.Dict["new"] != "hello again" {
		t.Fatalf("Config.new is not set")
	}

	// .. when a key is deleted:
	t.DoozerDel("/dict/bar")
	time.Sleep(100 * time.Millisecond)
	if _, ok := Config.Dict["bar"]; ok {
		t.Fatalf("Config.bar was not deleted")
	}

	fmt.Printf("%+v\n", Config)
}

// Test library

type DoozerConfigTest struct {
	Name string
	*testing.T
	doozer *doozer.Conn
	cfg    *DoozerConfig
	rev    int64
}

func NewDoozerConfigTest(name string, t *testing.T) *DoozerConfigTest {
	tt := DoozerConfigTest{name, t, nil, nil, 0}
	doozer, err := doozer.Dial("localhost:8046")
	if err != nil {
		t.Fatalf("cannot connect to doozerd: %s", err)
	}
	tt.doozer = doozer
	return &tt
}

func (t *DoozerConfigTest) Close() {
	t.doozer.Close()
}

func (t *DoozerConfigTest) Load(structValue interface{}, prefix string) {
	if t.cfg != nil {
		t.Fatalf("Load() was already called")
	}

	headRev, err := t.doozer.Rev()
	if err != nil {
		t.Fatal(err)
	}
	t.cfg = New(t.doozer, headRev, structValue, prefix)
	t.rev = headRev
	err = t.cfg.Load()
	if err != nil {
		t.Fatalf("failed to populate the struct from doozer: %s", err)
	}
	if t.cfg == nil {
		t.Fatalf("New returned nil")
	}
}

func (t *DoozerConfigTest) Monitor(pattern string) {
	if t.cfg == nil {
		t.Fatal("t.cfg is nil; Load() was not called?")
	}
	t.cfg.Monitor(pattern, func(name string, value interface{}, err error) {
		if err != nil {
			t.Fatalf("Monitor returned an error: %s", err)
		}
		// fmt.Printf("** mutation: %s == %s\n", name, value)
	})
}

func (t *DoozerConfigTest) DoozerSet(path, value string) {
	_, err := t.doozer.Set(path, 99999, []byte(value))
	if err != nil {
		t.Fatalf("failed to write to doozer: %s", err)
	}
}

func (t *DoozerConfigTest) DoozerDel(path string) {
	err := t.doozer.Del(path, 99999)
	if err != nil {
		t.Fatalf("failed to delete a file in doozer: %s", err)
	}
}
