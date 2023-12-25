package osm2ch

import (
	"github.com/paulmach/osm"
)

type Node struct {
	node osm.Node
	name string

	ID          osm.NodeID
	useCount    int
	controlType ControlType
	isCrossing  bool
	highway     string
}

type ControlType uint16

const (
	NOT_SIGNAL = ControlType(iota + 1)
	IS_SIGNAL
)

func (iotaIdx ControlType) String() string {
	return [...]string{"common", "signal"}[iotaIdx-1]
}
