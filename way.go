package osm2ch

import (
	"github.com/paulmach/osm"
)

type Way struct {
	ID     osm.WayID
	Oneway bool
	Nodes  osm.WayNodes
	TagMap osm.Tags
}