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
