package osm2ch

import (
	"fmt"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"
)

const (
	DEFAULT_FIRST_VERTEX = 0
	DEFAULT_FIRST_EDGE   = 0
)

type macroEdge struct {
	tagMap             osm.Tags
	geom               orb.LineString
	lengthMeters       float64
	linkConnectionType LinkConnectionType
	linkType           LinkType
	linkClass          LinkClass

	id              int
	osmID           osm.WayID
	osmSourceNodeID osm.NodeID
	osmTargetNodeID osm.NodeID
	wasOneWay       bool
	isCycle         bool
}

func (data *OSMData) prepareEdgesMacro(firstVertex, firstEdge int, verbose bool) ([]*macroEdge, error) {

	if verbose {
		fmt.Printf("Preparing edges for macroscopic graph...")
	}
	st := time.Now()

	macroEdges := make([]*macroEdge, 0, len(data.ways))

	edgesObserved := firstEdge
	for _, way := range data.ways {
		if way.isPOI() {
			// @todo: handle POI
			continue
		}

		if len(way.Nodes) < 2 {
			if verbose {
				fmt.Printf("\n\t[WARNING]: Way with %d nodes met. Way ID: '%d'\n", len(way.Nodes), way.ID)
			}
			continue
		}

		edge := &macroEdge{
			id:              edgesObserved,
			osmID:           way.ID,
			geom:            make(orb.LineString, 0, len(way.Nodes)),
			osmSourceNodeID: way.Nodes[0],
			osmTargetNodeID: way.Nodes[len(way.Nodes)-1],
			wasOneWay:       way.Oneway,
			isCycle:         false,
		}

		if edge.osmSourceNodeID == edge.osmTargetNodeID {
			edge.isCycle = true
		}
		if way.isHighway() {
			if way.isHighwayPOI() {
				// @todo: handle POI
				continue
			}
			// Ignore ways `area` tag provided
			if way.area != "" && way.area != "no" {
				continue
			}
			// Ignore ways of negligible types
			if way.isHighwayNegligible() {
				continue
			}
			if linkInfo, ok := linkTypeByHighway[getHighwayType(way.highway)]; ok {
				if way.OnewayDefault {
					// Apply default `oneway`` if it hasn't been defined yet
					edge.wasOneWay = onewayDefaultByLink[linkInfo.linkType]
				}
				edge.linkConnectionType = linkInfo.linkConnectionType
				edge.linkType = linkInfo.linkType
				edge.linkClass = LINK_CLASS_HIGHWAY
			} else {
				// if verbose {
				fmt.Printf("\n\t[WARNING]: Unhandled `highway` tag value: '%s'. Way ID: '%d'\n", way.highway, way.ID)
				continue
				// }
			}
		} else if way.isRailway() {
			// @todo: handle railways
			if way.isRailwayPOI() {
				// @todo: handle POI
				continue
			}
		} else if way.isAeroway() {
			// @todo: handle aeroways
			if way.isAerowayPOI() {
				// @todo: handle POI
				continue
			}
		} else {
			continue
		}

		for _, nodeID := range way.Nodes {
			if node, ok := data.nodes[nodeID]; ok {
				pt := orb.Point{node.node.Lon, node.node.Lat}
				edge.geom = append(edge.geom, pt)
			} else {
				return nil, fmt.Errorf("No such node %d", nodeID)
			}
		}
		edge.lengthMeters = geo.LengthHaversign(edge.geom)

		macroEdges = append(macroEdges, edge)
		edgesObserved++
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	return macroEdges, nil
}
