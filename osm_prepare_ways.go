package osm2ch

import (
	"fmt"
	"time"
)

func (data *OSMDataRaw) prepareWays(verbose bool) error {

	if verbose {
		fmt.Printf("Prepare ways...")
	}
	st := time.Now()

	data.waysMedium = make([]*WayData, 0, len(data.ways))
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
			// if way.lanes <= 0 {
			// 	if lanes, ok := defaultLanesByLinkType[way.linkType]; ok {
			// 		way.lanes = lanes
			// 	}
			// }

			// Need to consider allowed tags only
			allowedAgentTypes := way.getAllowableAgentType()
			agentsIntersection := agentsIntersection(allowedAgentTypes, data.allowedAgentTypes)
			if len(agentsIntersection) == 0 {
				continue
			}
			way.allowedAgentTypes = make([]AgentType, 0, len(agentsIntersection))
			for agentType := range agentsIntersection {
				way.allowedAgentTypes = append(way.allowedAgentTypes, agentType)
			}
			// way.geom = make(orb.LineString, 0, len(way.Nodes))
			data.nodes[way.Nodes[0]].isCrossing = true
			data.nodes[way.Nodes[len(way.Nodes)-1]].isCrossing = true
			for _, nodeID := range way.Nodes {
				if _, ok := data.nodes[nodeID]; ok {
					data.nodes[nodeID].useCount++
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

func (data *OSMDataRaw) markPureCycles(verbose bool) error {
	if verbose {
		fmt.Printf("Cook well-done ways...")
	}
	st := time.Now()
	for _, way := range data.waysMedium {
		// Find and mark pure cycles
		if way.isCycle {
			way.isPureCycle = true
			for _, nodeID := range way.Nodes {
				if _, ok := data.nodes[nodeID]; !ok {
					return fmt.Errorf("No such node '%d'. Way ID: '%d'", nodeID, way.ID)
				}
				if data.nodes[nodeID].isCrossing {
					way.isPureCycle = false
				}
			}
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return nil
}
