// Package classification Drone API.
//
//     Schemes: http, https
//     BasePath: /api
//     Version: 1.0.0
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package swagger

//go:generate swagger generate spec -o files/swagger.json
//go:generate go-bindata -pkg swagger -o swagger_gen.go files/
