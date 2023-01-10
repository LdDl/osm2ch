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

	/* Mesoscopic */
	movements        []*Movement
	movementIsNeeded bool
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

// genMovement generates Movement
func (node *NetworkNode) genMovement(movementID *MovementID, links map[NetworkLinkID]*NetworkLink) []*Movement {
	response := []*Movement{}

	if movementID == nil {
		return response
	}
	income := len(node.incomingLinks)
	outcome := len(node.outcomingLinks)
	if income == 0 || outcome == 0 {
		return response
	}

	if outcome == 1 {
		incomingLinksList := []*NetworkLink{}
		outcomingLinkID := node.outcomingLinks[0]
		outcomingLink, ok := links[outcomingLinkID]
		if ok {
			for _, incomingLinkID := range node.incomingLinks {
				if incomingLink, ok := links[incomingLinkID]; ok {
					if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
						incomingLinksList = append(incomingLinksList, incomingLink)
					}
				} else {
					return response
				}
			}
		}
		if len(incomingLinksList) == 0 {
			return response
		}

		connections := getSpansConnections(outcomingLink, incomingLinksList)
		for i, incomingLink := range incomingLinksList {
			incomeLaneStart := connections[i][0].first + 1
			incomeLaneEnd := connections[i][0].second + 1
			outcomeLaneStart := connections[i][1].first + 1
			outcomeLaneEnd := connections[i][1].second + 1
			lanesNum := incomeLaneEnd - incomeLaneStart + 1
			allowedAgentTypes := make([]AgentType, len(incomingLink.allowedAgentTypes))
			copy(allowedAgentTypes, incomingLink.allowedAgentTypes)
			mvmt := Movement{
				ID:                *movementID,
				NodeID:            node.ID,
				osmNodeID:         node.osmNodeID,
				IncomingLinkID:    incomingLink.ID,
				OutcomingLinkID:   outcomingLink.ID,
				incomeLaneStart:   incomeLaneStart,
				incomeLaneEnd:     incomeLaneEnd,
				outcomeLaneStart:  outcomeLaneStart,
				outcomeLaneEnd:    outcomeLaneEnd,
				lanesNum:          lanesNum,
				fromOsmNodeID:     incomingLink.sourceOsmNodeID,
				toOsmNodeID:       outcomingLink.targetOsmNodeID,
				controlType:       node.controlType,
				allowedAgentTypes: allowedAgentTypes,
			}
			mvmt.movementCompositeType, mvmt.movementType = movementBetweenLines(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
			mvmt.geom = movementGeomBetweenLines(incomingLink.geom, outcomingLink.geom)
			*movementID++
			response = append(response, &mvmt)
		}
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
						return response
					}
				}
				if len(outcomingLinksList) == 0 {
					return response
				}
				connections := getIntersectionsConnections(incomingLink, outcomingLinksList)
				for i, outcomingLink := range outcomingLinksList {
					incomeLaneStart := connections[i][0].first + 1
					incomeLaneEnd := connections[i][0].second + 1
					outcomeLaneStart := connections[i][1].first + 1
					outcomeLaneEnd := connections[i][1].second + 1
					lanesNum := incomeLaneEnd - incomeLaneStart + 1
					allowedAgentTypes := make([]AgentType, len(incomingLink.allowedAgentTypes))
					copy(allowedAgentTypes, incomingLink.allowedAgentTypes)

					mvmt := Movement{
						ID:                *movementID,
						NodeID:            node.ID,
						osmNodeID:         node.osmNodeID,
						IncomingLinkID:    incomingLinkID,
						OutcomingLinkID:   outcomingLink.ID,
						incomeLaneStart:   incomeLaneStart,
						incomeLaneEnd:     incomeLaneEnd,
						outcomeLaneStart:  outcomeLaneStart,
						outcomeLaneEnd:    outcomeLaneEnd,
						lanesNum:          lanesNum,
						fromOsmNodeID:     incomingLink.sourceOsmNodeID,
						toOsmNodeID:       outcomingLink.targetOsmNodeID,
						controlType:       node.controlType,
						allowedAgentTypes: allowedAgentTypes,
					}
					mvmt.movementCompositeType, mvmt.movementType = movementBetweenLines(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
					mvmt.geom = movementGeomBetweenLines(incomingLink.geom, outcomingLink.geom)
					*movementID++
					response = append(response, &mvmt)

				}
			} else {
				return response
			}
		}
	}
	node.movements = make([]*Movement, 0, len(response))
	node.movements = append(node.movements, response...)
	return response
}
