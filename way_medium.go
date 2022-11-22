package osm2ch

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

type WayMedium struct {
	tagMap             osm.Tags
	geom               orb.LineString
	lengthMeters       float64
	linkConnectionType LinkConnectionType
	linkType           LinkType
	linkClass          LinkClass
	lanesNum           int
	maxSpeed           float64
	freeSpeed          float64
	capacity           int

	id              int
	osmID           osm.WayID
	osmSourceNodeID osm.NodeID
	osmTargetNodeID osm.NodeID
	wasOneWay       bool
	isCycle         bool
}
