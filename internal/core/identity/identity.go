// Package identity defines verified actor references and the authentication port.
package identity

import (
	"context"
	"errors"
	"strings"
)

var ErrUnauthenticated = errors.New("authentication required")

type ActorRef struct {
	Source  string `json:"source"`
	Subject string `json:"subject"`
}

func (a ActorRef) Valid() bool {
	return a.Source != "" && a.Subject != "" && len(a.Source) <= 255 && len(a.Subject) <= 255 && !strings.ContainsAny(a.Source+a.Subject, "\x00\r\n") && strings.TrimSpace(a.Source) == a.Source && strings.TrimSpace(a.Subject) == a.Subject
}

type Authenticator interface {
	Authenticate(context.Context, string) (ActorRef, error)
}
