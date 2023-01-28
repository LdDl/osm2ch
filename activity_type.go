package osm2ch

import (
	"fmt"
)

type ActivityType uint16

const (
	ACTIVITY_POI = ActivityType(iota + 1)
	ACTIVITY_LINK
	ACTIVITY_NONE = ActivityType(0)
)

func (iotaIdx ActivityType) String() string {
	return [...]string{"none", "poi", "link"}[iotaIdx]
}

func (net *NetworkMacroscopic) genActivityType() error {
	nodesLinkTypesCounters := make(map[NetworkNodeID]map[LinkType]int)
	for _, link := range net.links {
		sourceNodeID := link.sourceNodeID
		if _, ok := net.nodes[sourceNodeID]; !ok {
			return fmt.Errorf("No source node with ID '%d'. Link ID: '%d'", sourceNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID]; !ok {
			nodesLinkTypesCounters[sourceNodeID] = make(map[LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[sourceNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[sourceNodeID][link.linkType]++
		}

		targetNodeID := link.targetNodeID
		if _, ok := net.nodes[targetNodeID]; !ok {
			return fmt.Errorf("No target node with ID '%d'. Link ID: '%d'", targetNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID]; !ok {
			nodesLinkTypesCounters[targetNodeID] = make(map[LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[targetNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[targetNodeID][link.linkType]++
		}
	}

	for nodeID, node := range net.nodes {
		if node.poiID > -1 {
			node.activityType = ACTIVITY_POI
			node.activityLinkType = LINK_UNDEFINED
		}
		if linkTypesCounters, ok := nodesLinkTypesCounters[nodeID]; ok {
			maxLinkType := LINK_UNDEFINED
			maxLinkTypeCount := 0
			for linkType, counter := range linkTypesCounters {
				if counter > maxLinkTypeCount {
					maxLinkTypeCount = counter
					maxLinkType = linkType
				}
			}
			// @TODO: What to do when thee are several link types with the same max count?
			if maxLinkType > 0 {
				node.activityType = ACTIVITY_LINK
				node.activityLinkType = maxLinkType
			} else {
				node.activityType = ACTIVITY_NONE
				node.activityLinkType = LINK_UNDEFINED
			}
		}
	}

	for _, node := range net.nodes {
		node.boundaryType = BOUNDARY_NONE
		if node.activityType == ACTIVITY_POI {
			continue
		}
		if len(node.outcomingLinks) == 0 {
			node.boundaryType = BOUNDARY_INCOME_ONLY
		} else if len(node.incomingLinks) == 0 {
			node.boundaryType = BOUNDARY_OUTCOME_ONLY
		} else if len(node.incomingLinks) == 1 && (len(node.outcomingLinks) == 1) {
			incomingLink, ok := net.links[node.incomingLinks[0]]
			if !ok {
				return fmt.Errorf("No incoming link with ID '%d'. Node ID: '%d'", node.incomingLinks[0], node.ID)
			}
			outcomingLink, ok := net.links[node.outcomingLinks[0]]
			if !ok {
				return fmt.Errorf("No incoming link with ID '%d'. Node ID: '%d'", node.outcomingLinks[0], node.ID)
			}
			if incomingLink.sourceNodeID == outcomingLink.targetNodeID {
				node.boundaryType = BOUNDARY_INCOME_OUTCOME
			}
		}
	}
	for _, node := range net.nodes {
		if node.boundaryType == BOUNDARY_NONE {
			continue
		}
		node.zoneID = node.ID
	}
	return nil
}
