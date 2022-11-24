package osm2ch

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

/* Nodes stuff */

type NetworkNodeID int

type NetworkNode struct {
	name           string
	osmHighway     string
	ID             NetworkNodeID
	osmNodeID      osm.NodeID
	intersectionID int
	controlType    ControlType
	geom           orb.Point
}

func networkNodeFromOSM(id NetworkNodeID, nodeOSM *Node) *NetworkNode {
	node := NetworkNode{
		name:           nodeOSM.name,
		osmHighway:     nodeOSM.highway,
		ID:             id,
		osmNodeID:      nodeOSM.ID,
		intersectionID: -1,
		controlType:    nodeOSM.controlType,
		geom:           nodeOSM.node.Point(),
	}
	return &node
}
