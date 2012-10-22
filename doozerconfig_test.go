// before running the test, install and run doozerd,
// https://github.com/ActiveState/doozerd/wiki/Instructions-to-compile-doozer

package doozerconfig

import (
	_ "fmt"
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
	t.Monitor()
	t.DoozerSet("/foo", "69")
	time.Sleep(100 * time.Millisecond)
	if Config.Foo != 69 {
		t.Fatalf("Config.Foo was not modified")
	}
}

// Test library

type DoozerConfigTest struct {
	Name string
	*testing.T
	doozer *doozer.Conn
	cfg    *DoozerConfig
}

func NewDoozerConfigTest(name string, t *testing.T) *DoozerConfigTest {
	tt := DoozerConfigTest{name, t, nil, nil}
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
	t.cfg = New(t.doozer, structValue, prefix)
	err := t.cfg.Load()
	if err != nil {
		t.Fatalf("failed to populate the struct from doozer: %s", err)
	}
	if t.cfg == nil {
		t.Fatalf("New returned nil")
	}
}

func (t *DoozerConfigTest) Monitor() {
	headRev, err := t.doozer.Rev()
	if err != nil {
		t.Fatal(err)
	}
	if t.cfg == nil {
		t.Fatal("t.cfg is nil; Load() was not called?")
	}
	go func() {
		t.cfg.Monitor("/foo*", headRev, func(name string, value interface{}, err error) {
			if err != nil {
				t.Fatalf("Monitor returned an error: %s", err)
			}
		})
	}()
}

func (t *DoozerConfigTest) DoozerSet(path, value string) {
	_, err := t.doozer.Set(path, 99999, []byte(value))
	if err != nil {
		t.Fatalf("failed to write to doozer: %s", err)
	}
}
