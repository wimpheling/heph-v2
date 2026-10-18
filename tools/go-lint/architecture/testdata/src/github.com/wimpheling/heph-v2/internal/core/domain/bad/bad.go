package bad

import (
	"context"
	"fmt"
	"os"
	"unapproved"

	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	op "github.com/wimpheling/heph-v2/internal/core/operation"
)

func comparisons(input a.ListInput) {
	isAdmin := input.Role == "admin" // want "domain computations belong"
	_ = isAdmin
	if input.Role == "admin" { // want "domain computations belong"
	}
	value := input.Count + 1 // want "domain computations belong"
	value++                  // want "domain computations belong"
	value += 1               // want "domain computations belong"
	_ = value
	_ = input.Name[0]  // want "domain computations belong"
	_ = input.Name[:1] // want "domain computations belong"
}
func wrongError(err *int) {
	if err != nil { // want "domain computations belong"
	}
}
func iteration() {
	for { // want "loops, switches"
	}
	for range []int{} { // want "loops, switches"
	}
	switch true { // want "loops, switches"
	}
}
func calls(service *a.Service, ctx context.Context, input a.ListInput) {
	fmt.Println("bad")   // want "domain calls require"
	os.ReadFile("file")  // want "domain calls require"
	_ = len(input.Name)  // want "domain calls require"
	alias := service.Get // want "domain function references"
	alias(ctx, input)    // want "domain calls require"
	func() {}()          // want "domain calls require" "anonymous functions"
}
func Exported(ctx context.Context, actor identity.ActorRef, input a.ListInput, service *a.Service) (int, error) { // want "domain business handlers must be private"
	return service.Create(ctx, actor, input)
}

var leaked = Exported // want "raw domain handlers" "mutable global state"
func raw(ctx context.Context, actor identity.ActorRef, input a.ListInput, service *a.Service) (int, error) {
	return Exported(ctx, actor, input, service) // want "domain calls require" "raw domain handlers"
}
func BadFactory(service *a.Service) *op.Operation[a.ListInput, int] { // want "domain operation factories must only return"
	service.Get(context.Background(), a.ListInput{}) // want "domain calls require"
	return op.Define(service, op.Public[a.ListInput](), raw)
}
func MissingPolicy(service *a.Service) *op.Operation[a.ListInput, int] { // want "domain operation factories must only return"
	return op.Define(service, op.Policy[a.ListInput]{}, raw) // want "construct operations and policies"
}
func FakeAction(service *a.Service) *op.Operation[a.ListInput, int] {
	return op.Define(service, op.Require(a.Action("invented"), a.StoredScope), raw) // want "policies require a registered" "domain calls require"
}

var fake = op.Operation[a.ListInput, int]{} // want "construct operations and policies" "mutable global state"

func init() {} // want "unguarded initialization"

func globalWrite(input a.ListInput) {
	a.Shared = input   // want "domain assignments must target local variables"
	a.Shared.Count = 3 // want "domain assignments must target local variables"
}

func namedServiceIsNotApproved(service unapproved.Service) {
	service.Get() // want "domain calls require"
}
func concurrency(ch chan bool) {
	go fmt.Println("bad")    // want "loops, switches" "domain calls require"
	defer fmt.Println("bad") // want "loops, switches" "domain calls require"
	ch <- true               // want "loops, switches"
	select {                 // want "loops, switches"
	case <-ch: // want "domain computations belong"
	}
}
func typeSwitch(value any) {
	switch value.(type) { // want "loops, switches" "domain computations belong"
	case bool:
	}
}
