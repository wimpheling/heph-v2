package core

import (
	"net/http" // want "keep core independent of HTTP"

	"connectrpc.com/connect"                                     // want "keep core independent of HTTP"
	transport "github.com/wimpheling/heph-v2/gen/heph/system/v1" // want "keep core independent of HTTP"
)

var handler http.Handler
var response connect.Response
var request transport.Request
