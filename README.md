# GoKit CLI
This project is a more advanced version of [gk](https://github.com/kujtimiihoxha/gk).
The goal of the gokit cli is to be a tool that you can use while you develop your microservices with `gokit`.

While `gk` did help you create your basic folder structure it was not really able to be used further on in your project.
This is what `GoKit Cli` is aiming to change.


# Prerequisites 
GoKit Cli needs to ne installed using `go get` and `go install` so `Go` is a requirement to be able to test your services
[gokit](https://github.com/go-kit/kit) is needed.

# Table of Content
 - [Installation](#installation)
 - [Usage](#usage)
 - [Create a new service](#create-a-new-service)
 - [Generate the service](#generate-the-service)
 
# Installation
Before you install please read [prerequisites](#prerequisites)
```bash
go get github.com/kujtimiihoxha/kit
```
# Usage
```bash
kit help
```
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
- **Create the service boilerplate**: `hello/pkg/service/service.go`
- **Create the service middleware**: `hello/pkg/service/middleware.go`
- **Create the endpoint**:  `hello/pkg/endpoint/endpoint.go` and `hello/pkg/endpoint/endpoint_gen.go`
- **If using` --dmw` create the endpoint middleware**: `hello/pkg/endpoint/middleware.go`
- **Create the transport files e.x `http`**: `service-name/pkg/http/handler.go`
- **Create the service main file** :boom:   
`hello/cmd/service/service.go`  
`hello/cmd/service/service_gen.go`   
`hello/cmd/main.go`

**Notice** all the files that end with `_gen` will be regenerated when you add endpoints to your service and 
you rerun `kit g s hello` 