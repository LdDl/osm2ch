package osm2ch

import (
	"fmt"
	"time"

	"github.com/paulmach/orb"
	"github.com/pkg/errors"
)

const (
	DEFAULT_FIRST_VERTEX = 0
	DEFAULT_FIRST_EDGE   = 0
)

func (data *OSMDataRaw) prepareMedium(verbose bool) error {
	err := data.prepareWaysMedium(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't preprocess ways")
	}

	err = data.prepareNodesMedium(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't preprocess nodes")
	}

	return nil
}

func (data *OSMDataRaw) prepareWaysMedium(verbose bool) error {

	if verbose {
		fmt.Printf("Cook medium ways...")
	}
	st := time.Now()

	data.waysMedium = make([]*WayData, 0, len(data.waysRaw))
	for _, way := range data.waysRaw {
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

		way.osmSourceNodeID = way.Nodes[0]
		way.osmTargetNodeID = way.Nodes[len(way.Nodes)-1]

		if way.osmSourceNodeID == way.osmTargetNodeID {
			way.isCycle = true
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
			linkInfo, ok := linkTypeByHighway[getHighwayType(way.highway)]
			if !ok {
				if verbose {
					fmt.Printf("\n\t[WARNING]: Unhandled `highway` tag value: '%s'. Way ID: '%d'\n", way.highway, way.ID)
				}
				continue
			}
			if way.OnewayDefault {
				// Apply default `oneway`` if it hasn't been defined yet
				way.Oneway = onewayDefaultByLink[linkInfo.linkType]
			}
			way.linkConnectionType = linkInfo.linkConnectionType
			way.linkType = linkInfo.linkType
			way.linkClass = LINK_CLASS_HIGHWAY
			if way.lanes <= 0 {
				if lanes, ok := defaultLanesByLinkType[way.linkType]; ok {
					way.lanes = lanes
				}
			}
			way.geom = make(orb.LineString, 0, len(way.Nodes))
			data.nodes[way.Nodes[0]].isCrossing = true
			data.nodes[way.Nodes[len(way.Nodes)-1]].isCrossing = true
			for _, nodeID := range way.Nodes {
				if node, ok := data.nodes[nodeID]; ok {
					data.nodes[nodeID].useCount++
					pt := orb.Point{node.node.Lon, node.node.Lat}
					way.geom = append(way.geom, pt)
				} else {
					return fmt.Errorf("No such node '%d'. Way ID: '%d'", nodeID, way.ID)
				}
			}
			data.waysMedium = append(data.waysMedium, way)
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
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	return nil
}

func (data *OSMDataRaw) prepareNodesMedium(verbose bool) error {
	if verbose {
		fmt.Printf("Cook medium nodes...")
	}
	st := time.Now()
	for nodeID, nodeMedium := range data.nodes {
		if nodeMedium.useCount >= 2 || nodeMedium.controlType == IS_SIGNAL {
			data.nodes[nodeID].isCrossing = true
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return nil
}
