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

	/* Not used */
	isCentroid bool
}

func networkNodeFromOSM(id NetworkNodeID, nodeOSM *Node) *NetworkNode {
	node := NetworkNode{
		incomingLinks:    make([]NetworkLinkID, 0),
		outcomingLinks:   make([]NetworkLinkID, 0),
		activityType:     ACTIVITY_NONE,
		name:             nodeOSM.name,
		osmHighway:       nodeOSM.highway,
		ID:               id,
		osmNodeID:        nodeOSM.ID,
		intersectionID:   -1,
		zoneID:           -1,
		poiID:            -1,
		controlType:      nodeOSM.controlType,
		boundaryType:     BOUNDARY_NONE,
		geom:             nodeOSM.node.Point(),
		movementIsNeeded: true, // Consider all nodes as intersections by default
	}
	return &node
}

// genMovement generates set of movement for given node
func (node *NetworkNode) genMovement(movementID *MovementID, links map[NetworkLinkID]*NetworkLink) []*Movement {
	movements := []*Movement{}

	if movementID == nil {
		return movements
	}
	income := len(node.incomingLinks)
	outcome := len(node.outcomingLinks)
	if income == 0 || outcome == 0 {
		return movements
	}

	if outcome == 1 {
		// Merge
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
					return movements
				}
			}
		}
		if len(incomingLinksList) == 0 {
			return movements
		}

		connections := getSpansConnections(outcomingLink, incomingLinksList)

		incomingLaneIndices := outcomingLink.GetOutcomingLaneIndices()

		for i, incomingLink := range incomingLinksList {
			incomeLaneIndexStart := connections[i][0].first
			incomeLaneIndexEnd := connections[i][0].second
			outcomeLaneIndexStart := connections[i][1].first
			outcomeLaneIndexEnd := connections[i][1].second
			lanesNum := incomeLaneIndexEnd - incomeLaneIndexStart + 1
			allowedAgentTypes := make([]AgentType, len(incomingLink.allowedAgentTypes))
			copy(allowedAgentTypes, incomingLink.allowedAgentTypes)

			outcomingLaneIndices := incomingLink.GetOutcomingLaneIndices()
			mvmt := Movement{
				ID:                    *movementID,
				NodeID:                node.ID,
				osmNodeID:             node.osmNodeID,
				IncomingLinkID:        incomingLink.ID,
				OutcomingLinkID:       outcomingLink.ID,
				startIncomeLaneSeqID:  incomeLaneIndexStart,
				endIncomeLaneSeqID:    incomeLaneIndexEnd,
				startOutcomeLaneSeqID: outcomeLaneIndexStart,
				endOutcomeLaneSeqID:   outcomeLaneIndexEnd,
				incomeLaneStart:       outcomingLaneIndices[incomeLaneIndexStart],
				incomeLaneEnd:         outcomingLaneIndices[incomeLaneIndexEnd],
				outcomeLaneStart:      incomingLaneIndices[outcomeLaneIndexStart],
				outcomeLaneEnd:        incomingLaneIndices[outcomeLaneIndexEnd],
				lanesNum:              lanesNum,
				fromOsmNodeID:         incomingLink.sourceOsmNodeID,
				toOsmNodeID:           outcomingLink.targetOsmNodeID,
				controlType:           node.controlType,
				allowedAgentTypes:     allowedAgentTypes,
			}
			mvmt.movementCompositeType, mvmt.movementType = movementBetweenLines(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
			mvmt.geom = movementGeomBetweenLines(incomingLink.geom, outcomingLink.geom)
			*movementID++
			movements = append(movements, &mvmt)
		}
	} else {
		// Diverge
		// Intersections
		for _, incomingLinkID := range node.incomingLinks {
			if incomingLink, ok := links[incomingLinkID]; ok {
				outcomingLinksList := []*NetworkLink{}
				for _, outcomingLinkID := range node.outcomingLinks {
					if outcomingLink, ok := links[outcomingLinkID]; ok {
						if incomingLink.sourceNodeID != outcomingLink.targetNodeID { // Ignore all reverse directions
							outcomingLinksList = append(outcomingLinksList, outcomingLink)
						}
					} else {
						return movements
					}
				}
				if len(outcomingLinksList) == 0 {
					return movements
				}

				connections := getIntersectionsConnections(incomingLink, outcomingLinksList)

				outcomingLaneIndices := incomingLink.GetOutcomingLaneIndices()

				for i, outcomingLink := range outcomingLinksList {
					incomeLaneIndexStart := connections[i][0].first
					incomeLaneIndexEnd := connections[i][0].second
					outcomeLaneIndexStart := connections[i][1].first
					outcomeLaneIndexEnd := connections[i][1].second
					lanesNum := incomeLaneIndexEnd - incomeLaneIndexStart + 1
					allowedAgentTypes := make([]AgentType, len(incomingLink.allowedAgentTypes))
					copy(allowedAgentTypes, incomingLink.allowedAgentTypes)

					incomingLaneIndices := outcomingLink.GetOutcomingLaneIndices()
					mvmt := Movement{
						ID:                    *movementID,
						NodeID:                node.ID,
						osmNodeID:             node.osmNodeID,
						IncomingLinkID:        incomingLink.ID,
						OutcomingLinkID:       outcomingLink.ID,
						startIncomeLaneSeqID:  incomeLaneIndexStart,
						endIncomeLaneSeqID:    incomeLaneIndexEnd,
						startOutcomeLaneSeqID: outcomeLaneIndexStart,
						endOutcomeLaneSeqID:   outcomeLaneIndexEnd,
						incomeLaneStart:       outcomingLaneIndices[incomeLaneIndexStart],
						incomeLaneEnd:         outcomingLaneIndices[incomeLaneIndexEnd],
						outcomeLaneStart:      incomingLaneIndices[outcomeLaneIndexStart],
						outcomeLaneEnd:        incomingLaneIndices[outcomeLaneIndexEnd],
						lanesNum:              lanesNum,
						fromOsmNodeID:         incomingLink.sourceOsmNodeID,
						toOsmNodeID:           outcomingLink.targetOsmNodeID,
						controlType:           node.controlType,
						allowedAgentTypes:     allowedAgentTypes,
					}
					mvmt.movementCompositeType, mvmt.movementType = movementBetweenLines(incomingLink.geomEuclidean, outcomingLink.geomEuclidean)
					mvmt.geom = movementGeomBetweenLines(incomingLink.geom, outcomingLink.geom)
					*movementID++
					movements = append(movements, &mvmt)

				}
			} else {
				return movements
			}
		}
	}
	node.movements = make([]*Movement, 0, len(movements))
	node.movements = append(node.movements, movements...)
	return movements
}
