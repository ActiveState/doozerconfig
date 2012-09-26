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
    RedisUri         string // not in doozer
    Verbose          bool   // not in doozer
}

func init() {
    doozer, err := doozer.Dial(getDoozerURI())
    if err != nil {
        log.Fatal(err)
    }

    headRev, err := doozer.Rev()
    if err != nil {
        log.Fatal(err)
    }

    cfg := doozerconfig.New(doozer, &Config, "/proc/")
    err = cfg.Load()  // this populates the Config struct
    if err != nil {
        log.Fatal(err)
    }
    Config.RedisUri = fmt.Sprintf("%s:%d", Config.RedisHost, Config.RedisPort)

    // watch for live changes to doozer config
    go func() {
        cfg.Monitor("/proc/logyard/config/*", headRev, func(name string, value interface{}, err error) {
            if err != nil {
                log.Fatal(err)
            }
            log.Printf("config changed in doozer; %s=%v\n", name, value)            
        }) 
    }()
}
```

## Installation

```bash
$ go get github.com/srid/doozerconfig
```
