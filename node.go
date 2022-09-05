package osm2ch

import (
	"github.com/paulmach/osm"
)

type Node struct {
	ID       osm.NodeID
	useCount int
	node     osm.Node
}
