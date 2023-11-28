package spec

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// TODO: review
type Epoch struct {
	Epoch                 phase0.Epoch
	Slot                  phase0.Slot
	NumAttestations       int
	NumAttValidators      int
	NumValidators         int
	TotalBalance          float32
	AttEffectiveBalance   float32
	TotalEffectiveBalance float32
	MissingSource         int
	MissingTarget         int
	MissingHead           int
	Timestamp             int64
	NumSlashed            int
	NumActive             int
	NumExit               int
	NumInActivation       int
}

func (f Epoch) Type() ModelType {
	return EpochModel
}
