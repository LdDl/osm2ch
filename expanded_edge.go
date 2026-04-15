package osm2ch

import (
	"github.com/LdDl/go-gmns/gmns"
	"github.com/paulmach/osm"
)

// ExpandedEdge represents an edge in the edge-expanded graph.
// Source and Target are IDs of the original macro links (edges).
type ExpandedEdge struct {
	ID              int64
	Source          EdgeID
	Target          EdgeID
	SourceOSMWayID  osm.WayID
	TargetOSMWayID  osm.WayID
	SourceComponent expandedEdgeComponent
	TargeComponent  expandedEdgeComponent
	WasOneway       bool
	LengthMeters    float64
	FreeSpeed       float64
	Geom            []GeoPoint
}

// expandedEdgeComponent represents source/target nodes of the original macro link
type expandedEdgeComponent struct {
	// macro graph source node ID
	SourceNodeID gmns.NodeID
	// macro graph target node ID
	TargetNodeID gmns.NodeID
	// original OSM source node ID
	SourceOSMNodeID osm.NodeID
	// original OSM target node ID
	TargetOSMNodeID osm.NodeID
}
