// Package identity implements a configured, constant-time token identity source.
package identity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	core "github.com/wimpheling/heph-v2/internal/core/identity"
)

type Credential struct {
	Source  string `json:"source"`
	Subject string `json:"subject"`
	Token   string `json:"token"`
}
type entry struct {
	actor core.ActorRef
	hash  [32]byte
}
type Driver struct{ entries []entry }

func New(credentials []Credential) (*Driver, error) {
	if len(credentials) == 0 {
		return nil, fmt.Errorf("configure at least one bootstrap identity")
	}
	driver := &Driver{}
	seen := map[[32]byte]bool{}
	for _, c := range credentials {
		actor := core.ActorRef{Source: c.Source, Subject: c.Subject}
		if !actor.Valid() || len(c.Token) < 24 {
			return nil, fmt.Errorf("bootstrap identities require valid references and tokens of at least 24 characters")
		}
		hash := sha256.Sum256([]byte(c.Token))
		if seen[hash] {
			return nil, fmt.Errorf("bootstrap identity tokens must be distinct")
		}
		seen[hash] = true
		driver.entries = append(driver.entries, entry{actor: actor, hash: hash})
	}
	return driver, nil
}
func (d *Driver) Authenticate(ctx context.Context, token string) (core.ActorRef, error) {
	if err := ctx.Err(); err != nil {
		return core.ActorRef{}, err
	}
	hash := sha256.Sum256([]byte(token))
	var actor core.ActorRef
	matched := 0
	for _, entry := range d.entries {
		match := subtle.ConstantTimeCompare(hash[:], entry.hash[:])
		if match == 1 {
			actor = entry.actor
		}
		matched |= match
	}
	if matched != 1 {
		return core.ActorRef{}, core.ErrUnauthenticated
	}
	return actor, nil
}
