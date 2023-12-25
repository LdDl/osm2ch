package osm2ch

import (
	"github.com/paulmach/orb"
)

type NetworkLinkMesoscopic struct {
	geom          orb.LineString
	geomEuclidean orb.LineString
	lanesNum      int
	lanesChange   [2]int

	ID NetworkLinkID

	sourceNodeID NetworkNodeID // Corresponds to ID of mesoscopic node (not to macro or OSM)
	targetNodeID NetworkNodeID // Corresponds to ID of mesoscopic node (not to macro or OSM)

	macroLinkID NetworkLinkID
	macroNodeID NetworkNodeID

	/* Movement information */
	movementID                    MovementID
	movementCompositeType         MovementCompositeType // Inherited from movement
	movementLinkIncome            NetworkLinkID
	movementLinkOutcome           NetworkLinkID
	movementIncomeLaneStartSeqID  int
	movementOutcomeLaneStartSeqID int

	controlType       ControlType // Inherited from macroscopic node
	linkType          LinkType    // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	freeSpeed         float64     // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	capacity          int         // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node
	allowedAgentTypes []AgentType // Inherited either from macroscopic link or from first incoming incident edge in macroscopic node

	lengthMeters float64
	isConnection bool

	/* Microscopic */
	microNodesPerLane  [][]NetworkNodeID
	microNodesBikeLane []NetworkNodeID
	microNodesWalkLane []NetworkNodeID
}
