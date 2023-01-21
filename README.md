# Shortloop SDK for Go

`shortloop-go` provides client implementation of Shortloop SDK for the Go programming
language.

## Requirements

Current SDK version requires Go 1.17 or higher.  
For Gin framework, it requires gin version 1.4.0 or higher.

## Installation

`shortloop-go` can be installed like any other Go library through `go get`:

```console
$ go get github.com/short-loop/shortloop-go@latest
```
## Usage

1. Import sdk in your code

    For gin:
    ```Go
    import "github.com/short-loop/shortloop-go/shortloopgin"
    ```
   
    For mux:
    ```Go
    import "github.com/short-loop/shortloop-go/shortloopmux"
    ```

5. Initialize the sdk  
   Example for gin:
    ```Go
    router := gin.Default()
    sdk, err := shortloopgin.Init(shortloopgin.Options{
        ShortloopEndpoint: "http://localhost:8080",
        ApplicationName:   "test-service-go",
        LoggingEnabled:    true,
        LogLevel:          "INFO",
    })
    if err != nil {
        fmt.Println("Error initializing shortloopgin: ", err)
    } else {
        router.Use(sdk.Filter())
    }
    ```
   Example for mux:
    ```Go
    mux := mux.NewRouter()
    sdk, err := shortloopmux.Init(shortloopmux.Options{
        ShortloopEndpoint: "http://localhost:8080",
        ApplicationName:   "test-service-go",
        LoggingEnabled:    true,
        LogLevel:          "INFO",
    })
    if err != nil {
        fmt.Println("Error initializing shortloopmux: ", err)
    } else {
        mux.Use(sdk.Filter)
    }
    ```

