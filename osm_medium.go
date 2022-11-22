package osm2ch

import (
	"fmt"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
)

const (
	DEFAULT_FIRST_VERTEX = 0
	DEFAULT_FIRST_EDGE   = 0
)

type OSMDataMedium struct {
	ways []*WayMedium
}

func (data *OSMDataRaw) prepareWaysMedium(verbose bool) (*OSMDataMedium, error) {

	if verbose {
		fmt.Printf("Cook medium ways...")
	}
	st := time.Now()

	waysMedium := make([]*WayMedium, 0, len(data.ways))

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

		edge := &WayMedium{
			osmID:           way.ID,
			geom:            make(orb.LineString, 0, len(way.Nodes)),
			osmSourceNodeID: way.Nodes[0],
			osmTargetNodeID: way.Nodes[len(way.Nodes)-1],
			wasOneWay:       way.Oneway,
			lanesNum:        -1,
			maxSpeed:        -1,
			freeSpeed:       -1,
			capacity:        -1,
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

				if way.lanes > 0 {
					edge.lanesNum = way.lanes
				} else {
					if lanes, ok := defaultLanesByLinkType[edge.linkType]; ok {
						edge.lanesNum = lanes
					}
				}
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
		edge.prepareFlowParams()
		waysMedium = append(waysMedium, edge)
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	return &OSMDataMedium{
		ways: waysMedium,
	}, nil
}

var (
	defaultLanesByLinkType = map[LinkType]int{
		LINK_MOTORWAY:     4,
		LINK_TRUNK:        3,
		LINK_PRIMARY:      3,
		LINK_SECONDARY:    2,
		LINK_TERTIARY:     2,
		LINK_RESIDENTIAL:  1,
		LINK_SERVICE:      1,
		LINK_CYCLEWAY:     1,
		LINK_FOOTWAY:      1,
		LINK_TRACK:        1,
		LINK_UNCLASSIFIED: 1,
		LINK_CONNECTOR:    2,
	}
	defaultSpeedByLinkType = map[LinkType]float64{
		LINK_MOTORWAY:     120,
		LINK_TRUNK:        100,
		LINK_PRIMARY:      80,
		LINK_SECONDARY:    60,
		LINK_TERTIARY:     40,
		LINK_RESIDENTIAL:  30,
		LINK_SERVICE:      30,
		LINK_CYCLEWAY:     5,
		LINK_FOOTWAY:      5,
		LINK_TRACK:        30,
		LINK_UNCLASSIFIED: 30,
		LINK_CONNECTOR:    120,
	}
	defaultCapacityByLinkType = map[LinkType]int{
		LINK_MOTORWAY:     2300,
		LINK_TRUNK:        2200,
		LINK_PRIMARY:      1800,
		LINK_SECONDARY:    1600,
		LINK_TERTIARY:     1200,
		LINK_RESIDENTIAL:  1000,
		LINK_SERVICE:      800,
		LINK_CYCLEWAY:     800,
		LINK_FOOTWAY:      800,
		LINK_TRACK:        800,
		LINK_UNCLASSIFIED: 800,
		LINK_CONNECTOR:    9999,
	}
)

func (edge *WayMedium) prepareFlowParams() {
	if edge.capacity < 0 {
		if defaultCap, ok := defaultCapacityByLinkType[edge.linkType]; ok {
			edge.capacity = defaultCap
		}
	}
	if edge.freeSpeed < 0 {
		if edge.maxSpeed >= 0 {
			edge.freeSpeed = edge.maxSpeed
		} else {
			if defaultSpeed, ok := defaultSpeedByLinkType[edge.linkType]; ok {
				edge.freeSpeed = defaultSpeed
				edge.maxSpeed = defaultSpeed
			}
		}
	}
}
