package osm2ch

import (
	"fmt"
	"time"
)

func (data *OSMDataRaw) prepareWays(verbose bool, poi bool) error {

	if verbose {
		fmt.Printf("Prepare ways...")
	}
	st := time.Now()

	data.waysMedium = make([]*WayData, 0, len(data.ways))
	data.waysPOI = make([]*WayData, 0, len(data.ways)/2)

	for _, way := range data.ways {
		if way.isPOI() {
			data.waysPOI = append(data.waysPOI, way)
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
				way.wayPOI = way.highway
				data.waysPOI = append(data.waysPOI, way)
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
			fmt.Println("wwwww railways")
			// @TODO: handle railways
			if way.isRailwayPOI() {
				way.wayPOI = way.railway
				data.waysPOI = append(data.waysPOI, way)
				continue
			}
		} else if way.isAeroway() {
			// @TODO: handle aeroways
			fmt.Println("wwwww aeroways")
			if way.isAerowayPOI() {
				way.wayPOI = way.aeroway
				data.waysPOI = append(data.waysPOI, way)
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
		fmt.Printf("Inspect pure cycles...")
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
