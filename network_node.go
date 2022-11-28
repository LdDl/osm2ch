package osm2ch

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

/* Nodes stuff */

type NetworkNodeID int

type NetworkNode struct {
	incomingLinks    []NetworkLinkID
	outcomingLinks   []NetworkLinkID
	name             string
	osmHighway       string
	ID               NetworkNodeID
	osmNodeID        osm.NodeID
	intersectionID   int
	zoneID           NetworkNodeID
	poiID            PoiID
	controlType      ControlType
	boundaryType     BoundaryType
	activityType     ActivityType
	activityLinkType LinkType
	geom             orb.Point
	geomEuclidean    orb.Point
}

func networkNodeFromOSM(id NetworkNodeID, nodeOSM *Node) *NetworkNode {
	node := NetworkNode{
		incomingLinks:  make([]NetworkLinkID, 0),
		outcomingLinks: make([]NetworkLinkID, 0),
		activityType:   ACTIVITY_NONE,
		name:           nodeOSM.name,
		osmHighway:     nodeOSM.highway,
		ID:             id,
		osmNodeID:      nodeOSM.ID,
		intersectionID: -1,
		zoneID:         -1,
		poiID:          -1,
		controlType:    nodeOSM.controlType,
		boundaryType:   BOUNDARY_NONE,
		geom:           nodeOSM.node.Point(),
	}
	return &node
}

type MovementID int

// genMovement generates Movement
func (node *NetworkNode) genMovement(movementID MovementID, links map[NetworkLinkID]*NetworkLink) bool {
	income := len(node.incomingLinks)
	outcome := len(node.outcomingLinks)
	if income == 0 || outcome == 0 {
		return false
	}
	if outcome == 1 {
		incomingLinksList := []NetworkLinkID{}
		outcomingLinkID := node.outcomingLinks[0]
		if outcomingLink, ok := links[outcomingLinkID]; ok {
			for _, incomingLinkID := range node.incomingLinks {
				if incomingLink, ok := links[incomingLinkID]; ok {
					if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
						incomingLinksList = append(incomingLinksList, incomingLinkID)
					}
				} else {
					return false
				}
			}
		}
		if len(incomingLinksList) == 0 {
			return false
		}
		// @todo: handle
	} else {
		for _, incomingLinkID := range node.incomingLinks {
			if incomingLink, ok := links[incomingLinkID]; ok {
				outcomingLinksList := []*NetworkLink{}
				for _, outcomingLinkID := range node.outcomingLinks {
					if outcomingLink, ok := links[outcomingLinkID]; ok {
						if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
							outcomingLinksList = append(outcomingLinksList, outcomingLink)
						}
					} else {
						return false
					}
				}
				if len(outcomingLinksList) == 0 {
					return false
				}
				// @todo: handle
				connections := getIntersectionsConnections(incomingLink, outcomingLinksList)
				_ = connections
			} else {
				return false
			}
		}
	}

	return true
}
