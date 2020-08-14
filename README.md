# Asana [![Build Status](https://travis-ci.org/goasana/asana.svg?branch=master)](https://travis-ci.org/goasana/asana) [![GoDoc](http://godoc.org/github.com/goasana/asana?status.svg)](http://godoc.org/github.com/goasana/asana) [![Go Report Card](https://goreportcard.com/badge/github.com/goasana/asana)](https://goreportcard.com/report/github.com/goasana/asana)


Asana is a framework based on [Asana](http://asana.me) (fork).

## Quick Start

#### Create `hello` directory, cd `hello` directory

    mkdir hello
    cd hello
 
#### Init module

    go mod init

#### Download and install

    go get github.com/goasana/asana@v1.13.0

#### Create file `hello.go`
```go
package main

import "github.com/goasana/asana"

func main(){
    asana.Run()
}
```
#### Build and run

    go build hello.go
    ./hello

#### Go to [http://localhost:8080](http://localhost:8080)

Congratulations! You've just built your first **Asana** app.

###### [asana-example](https://github.com/asana-dev/asana-example)

## Features
* YAML file default config
* Kubernetes Map, File, Etcd, Consul, etc. config providers.
* Redis, Memcached, ssdb, gCache and local cache.
* RESTful support.
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
