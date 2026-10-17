package nested

import "net/http/httptest" // want "keep core independent of HTTP"

var _ = httptest.NewRecorder
