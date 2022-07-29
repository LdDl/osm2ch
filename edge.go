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
	Geom         []GeoPoint
}

// findOutComingEdges returns IDs of edges for given OSM Way object
func findOutComingEdges(givenEdge Edge, edges []Edge) []EdgeID {
	result := []EdgeID{}
	for _, edge := range edges {
		if edge.SourceNodeID == givenEdge.TargetNodeID && edge.ID != givenEdge.ID {
			result = append(result, edge.ID)
		}
	}
	return result
}
