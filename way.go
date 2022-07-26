package osm2ch

import (
	"github.com/paulmach/osm"
)

type Way struct {
	ID     int64
	Oneway bool
	Nodes  osm.WayNodes
	TagMap osm.Tags
}
