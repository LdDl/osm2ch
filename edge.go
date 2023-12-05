package osm2ch

import (
	"github.com/paulmach/osm"
)

type EdgeID int64

type Edge struct {
	ID           EdgeID
	WayID        osm.WayID
	SourceNodeID osm.NodeID
	TargetNodeID osm.NodeID
	WasOneway    bool
	CostMeters   float64
	/* CostSeconds  float64 */ //@todo: consider cost customization
	Geom                       []GeoPoint
}
