package osm2ch

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

type WayMedium struct {
	geom   orb.LineString
	tagMap osm.Tags

	osmTargetNodeID osm.NodeID
	osmSourceNodeID osm.NodeID
	osmID           osm.WayID

	lanesNum     int
	capacity     int
	freeSpeed    float64
	maxSpeed     float64
	lengthMeters float64

	linkClass          LinkClass
	linkType           LinkType
	linkConnectionType LinkConnectionType

	wasOneWay bool
	isCycle   bool
}
