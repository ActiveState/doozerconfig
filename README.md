# doozerconfig

## What Is It?

doozerconfig is a Go package for managing json-encoded configuration in doozer. Configuration is directly loaded into a Go struct (see example below). Using struct tags to define the doozer path, the package will automatically read, json-decode and assign the values to the corresponding struct fields. You can also watch for future changes to doozer config and have the struct automatically update.

In future, this package will also provide a way to write configuration back to doozer.

## Example

```Go
var Config struct {
    MaxRecordSize    int    `doozer:"logyard/config/max_record_size"`
    MaxRecordsPerApp int    `doozer:"logyard/config/max_records_per_app"`
    NatsUri          string `doozer:"cloud_controller/config/mbus"`
    RedisHost        string `doozer:"cloud_controller/config/redis/host"`
    RedisPort        int    `doozer:"cloud_controller/config/redis/port"`
    Verbose          bool   // not in doozer
}

func init() {
    doozer, err := doozer.Dial("localhost:8046")
    
    // map Config fields to "/proc/" + the struct tag above.
    // eg: MaxRecordSize will be mapped to /proc/logyard/config/max_record_size
    cfg := doozerconfig.New(doozer, &Config, "/proc/")

    // load config values from doozer and assign to Config fields
    err = cfg.Load()  

    // watch for live changes to doozer config
    go func() {
        // Monitor automatically updates the fields of `Config`; 
        // the callback function is called for every update or error.
        headRev, err := doozer.Rev()
        cfg.Monitor("/proc/logyard/config/*", headRev, 
                    func(name string, value interface{}, err error) {
            fmt.Printf("config changed in doozer; %s=%v\n", name, value)            
        }) 
    }()
}
```

## Installation

```bash
$ go get github.com/srid/doozerconfig
```
