# Asana [![Build Status](https://travis-ci.org/goasana/framework.svg?branch=master)](https://travis-ci.org/goasana/framework) [![GoDoc](http://godoc.org/github.com/goasana/framework?status.svg)](http://godoc.org/github.com/goasana/framework) [![Go Report Card](https://goreportcard.com/badge/github.com/goasana/framework)](https://goreportcard.com/report/github.com/goasana/framework)


Asana is clone of [beego](http://beego.me).

## Quick Start

#### Download and install

    go get github.com/goasana/framework

#### Create file `hello.go`
```go
package main

import "github.com/goasana/framework"

func main(){
    asana.Run()
}
```
#### Build and run

    go build hello.go
    ./hello

#### Go to [http://localhost:8080](http://localhost:8080)

Congratulations! You've just built your first **Asana** app.

## Features
* YAML file default config
* Kubernetes Map, File, Consul config providers.
* RESTful support
* Good for Microservices, RESTful protocol buffers with [gogo/protobuf](https://github.com/gogo/protobuf)
* MVC architecture
* Modularity
* Auto API documents
* Annotation router
* Namespace
* Powerful development tools
* Full stack for Web & API

## License

Asana source code is licensed under the Apache Licence, Version 2.0
(http://www.apache.org/licenses/LICENSE-2.0.html).
