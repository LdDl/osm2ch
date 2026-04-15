package osm2ch

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/paulmach/osm"
)

type EdgeID int64

type Edge struct {
	ID           EdgeID
	WayID        osm.WayID
	SourceNodeID gmns.NodeID // macro graph node ID
	TargetNodeID gmns.NodeID // macro graph node ID
	SourceOSMNodeID osm.NodeID
	TargetOSMNodeID osm.NodeID
	WasOneway    bool
	LengthMeters float64
	FreeSpeed    float64
	Geom         []GeoPoint
}
