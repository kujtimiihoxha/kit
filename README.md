# GoKit CLI  [![Build Status](https://travis-ci.org/kujtimiihoxha/kit.svg?branch=master)](https://travis-ci.org/kujtimiihoxha/kit)[![Go Report Card](https://goreportcard.com/badge/github.com/kujtimiihoxha/kit)](https://goreportcard.com/report/github.com/kujtimiihoxha/kit)[![Coverage Status](https://coveralls.io/repos/github/kujtimiihoxha/kit/badge.svg?branch=master)](https://coveralls.io/github/kujtimiihoxha/kit?branch=master)
This project is a more advanced version of [gk](https://github.com/kujtimiihoxha/gk).
The goal of the gokit cli is to be a tool that you can use while you develop your microservices with `gokit`.

While `gk` did help you create your basic folder structure it was not really able to be used further on in your project.
This is what `GoKit Cli` is aiming to change.


# Prerequisites 
GoKit Cli needs to be installed using `go get` and `go install` so `Go` is a requirement to be able to test your services
[gokit](https://github.com/go-kit/kit) is needed.

# Table of Content
 - [Installation](#installation)
 - [Usage](#usage)
 - [Create a new service](#create-a-new-service)
 - [Generate the service](#generate-the-service)
 - [Generate the client library](#generate-the-client-library)
 - [Generate new middlewares](#generate-new-middleware)
 - [Mod feature support](#mod-feature-support)
 - [Enable docker integration](#enable-docker-integration)
 
# Installation
Before you install please read [prerequisites](#prerequisites)
```bash
go get github.com/kujtimiihoxha/kit
```
# Usage
```bash
kit help
```

Also read this [medium story](https://medium.com/@kujtimii.h/creating-a-todo-app-using-gokit-cli-20f066a58e1)
# Create a new service
```bash
kit new service hello
kit n s hello # using aliases
```
This will generate the initial folder structure and the service interface

`service-name/pkg/service/service.go`
```go
package service

// HelloService describes the service.
type HelloService interface {
	// Add your methods here
	// e.x: Foo(ctx context.Context,s string)(rs string, err error)
}
```

# Generate the service
```bash
kit g s hello
kit g s hello --dmw # to create the default middleware
kit g s hello -t grpc # specify the transport (default is http)
```
This command will do these things:
- Create the service boilerplate: `hello/pkg/service/service.go`
- Create the service middleware: `hello/pkg/service/middleware.go`
- Create the endpoint:  `hello/pkg/endpoint/endpoint.go` and `hello/pkg/endpoint/endpoint_gen.go`
- If using` --dmw` create the endpoint middleware: `hello/pkg/endpoint/middleware.go`
- Create the transport files e.x `http`: `service-name/pkg/http/handler.go`
- Create the service main file :boom:   
`hello/cmd/service/service.go`  
`hello/cmd/service/service_gen.go`   
`hello/cmd/main.go`

:warning: **Notice** all the files that end with `_gen` will be regenerated when you add endpoints to your service and 
you rerun `kit g s hello` :warning: 

You can run the service by running:
```bash
go run hello/cmd/main.go
```

# Generate the client library
```bash
kit g c hello
```
This will generate the client library :sparkles: `http/client/http/http.go` that you can than use to call the service methods, you can use it like this:
```go
package main

import (
	"context"
	"fmt"

	client "hello/client/http"
	"github.com/go-kit/kit/transport/http"
)

func main() {
	svc, err := client.New("http://localhost:8081", map[string][]http.ClientOption{})
	if err != nil {
		panic(err)
	}

	r, err := svc.Foo(context.Background(), "hello")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Result:", r)
}
```
# Generate new middleware
```bash
kit g m hi -s hello
kit g m hi -s hello -e # if you want to add endpoint middleware
```
The only thing left to do is add your middleware logic and wire the middleware with your service/endpoint.
# Mod feature support
If you want to create project outside the gopath, you should use --mod_module flag when you create a new service, generate the service, the client library and the new middleware. The --mod_module value should be as same as your mod module path and is under your work directory. 

For example, under your work directory /XXX/github.com/groupname, running commands as follows:

```bash
kit n s hello --mod_mudole github.com/groupname/hello
kit g s hello --mod_mudole github.com/groupname/hello --dmw 
kit g c hello --mod_mudole github.com/groupname/hello
cd hello && go mod init github.com/groupname/hello
```
# Enable docker integration

```bash
kit g d
```
This will add the individual service docker files and one `docker-compose.yml` file that will allow you to start 
your services.
To start your services just run 
```bash
docker-compose up
```

After you run `docker-compose up` your services will start up and any change you make to your code will automatically
 rebuild and restart your service (only the service that is changed)