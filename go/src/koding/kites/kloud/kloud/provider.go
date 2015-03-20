package kloud

import (
	"koding/kites/kloud/machinestate"

	"golang.org/x/net/context"
)

type ctxKey int

const requestKey ctxKey = 0

// Provider is responsible of managing and controlling a cloud provider
type Provider interface {
	// Get returns a machine that should satisfy the necessary interfaces
	Get(id string) (interface{}, error)
}

type Builder interface {
	Build(ctx context.Context) error
}

type Stater interface {
	State() machinestate.State
}
