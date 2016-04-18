# Framework design

The goals are to be as unintrusive as possible. The swagger spec is the source of truth for your application.

The reference framework will make use of a swagger API that is based on the denco router.

The general idea is that it is a middleware which you provide with the swagger spec.
This document can be either JSON or YAML as both are supported.

In addition to the middleware there are some generator commands that will use the swagger spec to generate models, parameter models, operation interfaces and a mux.

## The middleware

Takes a raw spec document either as a []byte, and it adds the /api-docs route to serve this spec up.

The middleware performs validation, data binding and security as defined in the swagger spec.
It also uses the API to match request paths to functions of `func(paramsObject) (responseModel, error)`
The middleware does this by building up a series of rules for each operation. When the spec.Document is first created it analyzes the swagger spec and builds a routing, validation and binding rules for each operation in the specification. Before doing that it expands all the $ref fields in the swagger document. After expanding all the rules it validates the registrations made in the API and will take an configurable action for missing operations.

When a request comes in that doesn't match the /swagger.json endpoint it will look for it in the swagger spec routes.  It doesn't need to create a plan for anything anymore it did that at startup time but it will then execute the plan with the request and route params.
These are provided in the API. There is a tool to generate a statically typed API, based on operation names and operation interfaces

### The API

The reference API will use the denco router to register route handlers.
The actual request handler implementation is always the same.  The API must be designed in such a way that other frameworks can use their router implementation and perhaps their own validation infrastructure.

An API is served over http by a router, the default implementation is a router based on denco. This is just an interface implemenation so it can be replaced with another router should you so desire.

The API comes in 2 flavors an untyped one and a typed one.

#### Untyped API

The untyped API is the main glue. It takes registrations of operation ids to operation handlers.
It also takes the registrations for mime types to consumers and producers. And it links security schemes to authentication handlers.

```go
type OperationHandler func(interface{}) (interface{}, error)
```

The API has methods to register consumers, producers, auth handlers and operation handlers

The register consumer and producer methods are responsible for attaching extra serializers to media types. These are then used during content negotiation phases for look up and binding the data.

When an API is used to initialize a router it goes through a validation step.
This validation step will verify that all the operations in the spec have a handler registered to them.
It also ensures that for all the mentioned media types there are consumers and producers provided.
And it checks if for each authentication scheme there is a handler present.
If this is not the case it will exit the application with a non-zero exit code.

The register method takes an operation name and a swagger operation handler.  
It will then use that to build a path pattern for the router and it uses the swagger operation handler to produce a result based on the information in an incoming web request. It does this by injecing the handler in the swagger web request handler.

#### Typed API

The typed API uses a swagger spec to generate a typed API.

For this there is a generator that will take the swagger spec document.
It will then generate an interface for each operation and optionally a default implementation of that interface.
The default implemenation of an interface just returns a not implemented api error.

When all the interfaces and default implementations are generated it will generate a swagger mux implementation.
This swagger mux implemenation links all the interface implementations to operation names.

The typed API avoids reflection as much as possible, there are 1 or 2 validations that require it. For now it needs to include the swagger.json in the code for a few reasons.


### The request handler

The request handler does the following things:

1. Authenticate and authorize if necessary
2. Validate the request data
3. Bind the request data to the parameter struct based on the swagger schema
4. Validate the parameter struct based on the swagger schema
5. Produce a model or an error by invoking the operation interface
6. Create a response with status code etc based on the operation interface invocation result

#### Authentication

The authentication integration should execute security handlers. A security handler performs 2 functions it should authenticate and upon successful authentication it should authorize the request if the security scheme requires authorization. The authorization should be mainly required for the oauth2 based authentication flows.

```go
type Authenticator func(interface{}) (matched bool, principal interface{}, err error)
```

basic auth and api key type authorizations require the request for their authentication to work.

When we've determined a route matches we should check if the request is allowed to proceed.
To do this our middleware knows how to deal with basic auth and how to retrieve access tokens etc.
It does this by using the information in the security scheme object registered for a handler with the same scheme name.


#### Binding

Binding makes use of plain vanilla golang serializers and they are identified by the media type they consume and produce.

Binding is not only about request bodies but also about values obtained from headers, query string parameters and potentially the route path pattern. So the binding should make use of the full request object to produce a model.

It determines a serializer to use by looking in the the merged consumes values and the `Content-Type` header to determine which deserializer to use.  
When a result is produced it will do the same thing by making use of the `Accept` http header etc and the merged produces clauses for the operation endpoint.

```go
type RequestBinder interface {
  BindRequest(*http.Request, *router.MatchedRoute, swagger.Consumer) error
}
```

#### Validation

When the muxer registers routes it also builds a suite of validation plans, one for each operation.
Validation allows for adding custom validations for types through implementing a Validatable interface. This interface does not override but extends the validations provided by the swagger schema.

There is a mapping from validation name to status code, this mapping is also prioritized so that in the event of multiple validation errors that would required different status codes we get a consistent result. This prioritization can be done by the user by providing a ServeError function.

```go
type Validatable interface {
  Validate(strfmt.Registry) error
}

type Error struct {
  Code     int32
  Path     string
  In       string
  Value    interface{}
  Message  string
}
```

#### Execute operation

When all the middlewares have finished processing the request ends up in the operation handling middleware.  
This middleware is responsible for taking the bound model produced in the validation middleware and executing the registered operation handler for this request.
By the time the operation handler is executed we're sure the request is **authorized and valid**.

The result it gets from the operation handler will be turned into a response. Should the result of the operation handler be an error or a series of errors it will determine an appropriate status code and render the error result.


# Codegen

Codegen consists out of 2 parts. There is generating a server application from a swagger spec.
The second part is generating a swagger spec from go code based on annotations and information retrieved from the AST.

## Go generation

The goal of this code generation is to just take care of the boilerplate.
It uses a very small runtime to accommodate the swagger workflow. These are just small helpers for sharing some common
code.  The server application uses plain go http do do its thing. All other code is generated so you can read what it
does and if you think it's worth it.

The go server api generator however won't reuse those templates but define its own set, because currently no proper go support exists in that project. Once I'm happy with what they generate I'll contribute them back to the swagger-codegen project.

A generated client needs to have support for uploading files as multipart entries. The model generating code is shared between client and server. The things that operate with those models will be different.
A generated client could implement validation on the client side for the request parameters and received response. The meat of the client is not actually implemented as generated code but a single submit function that knows how to perform all the shared operations and then issue the request.
A client typically has only one consumer and producer registered. The content type for the request is the media type of the producer, the accept header is the media type of the consumer.

https://engineering.gosquared.com/building-better-api-docs
