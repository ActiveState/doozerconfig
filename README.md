# doozerconfig

## What Is It?

doozerconfig is a Go package for managing json-encoded configuration in doozer. Configuration in doozer is directly mapped to a Go struct. Configuration changes made in doozer are automatically reflected in the struct. For details, see the example below.

API documentation - http://go.pkgdoc.org/github.com/srid/doozerconfig

## Installation

```bash
$ go get github.com/srid/doozerconfig
```

## Example

```Go
var Config struct {
    MaxItems  int                `doozer:"config/max_items"`
    DbUri     string             `doozer:"config/db_uri"`
    EnvVars   map[string]string  `doozer:"config/envvars"`
    Verbose   bool   // not in doozer
}

func init() {
    doozer, _ := doozer.Dial("localhost:8046")
    rev, _ := doozer.Rev()
    
    // Map Config fields to "/myapp/" + the struct tag above.
    // eg: MaxItems will be mapped to /myapp/config/max_items
    //     EnvVars will be mapped to /myapp/config/envvars/*
    cfg := doozerconfig.New(doozer, rev, &Config, "/myapp/")

    // Load config values from doozer and assign to Config fields
    _ = cfg.Load()  

    // Watch for live changes to doozer config, and automatically
    // update the struct fields. The callback function can be used to
    // handle errors, and to get notified of changes.
    go cfg.Monitor("/myapp/config/*", func(name string, value interface{}, err error) {
        fmt.Printf("config changed in doozer; %s=%v\n", name, value)            
    })
}
```

# Notes

- If a file is deleted from doozer, the config struct is not updated
  (unless it a map entry). Perhaps we should update the value to a
  default value (as specified in a `default` struct tag).

- Writing configuration back to doozer is not supported yet.

# Changes

- **Oct 22, 2012**:

  - Support for loading map types

  - API: `Load` now takes doozer revision as a mandatory argument.

  - API: `Monitor` doesn't take a revision argument anymore. Instead
    it uses the one passed to `Load`.

- **Sep 25, 2012**:

  - Initial public release.
  