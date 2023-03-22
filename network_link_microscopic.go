package osm2ch

import (
	"github.com/paulmach/orb"
)

type MicroscopicLinkType uint16

const (
	LINK_FORWARD = MicroscopicLinkType(iota + 1)
	LINK_LANE_CHANGE
)

func (iotaIdx MicroscopicLinkType) String() string {
	return [...]string{"forward", "lane_change"}[iotaIdx-1]
}

type NetworkLinkMicroscopic struct {
	geom          orb.LineString
	geomEuclidean orb.LineString
	lengthMeters  float64

	ID NetworkLinkID

	sourceNodeID NetworkNodeID // Corresponds to ID of microscopic node (not to meso or macro or OSM)
	targetNodeID NetworkNodeID // Corresponds to ID of microscopic node (not to meso or macro or OSM)

	mesoLinkID  NetworkLinkID
	macroLinkID NetworkLinkID // Inherited from mesoscopic link
	macroNodeID NetworkNodeID // Inherited from mesoscopic link

	cellType              MicroscopicLinkType
	mesoLinkType          LinkType // Inherited from mesoscopic link
	freeSpeed             float64  // Inherited from mesoscopic link
	capacity              int      // Inherited from mesoscopic link
	additionalTravelCost  float64  // Penalty on movement through this link
	allowedAgentTypes     []AgentType
	firstMovement         bool
	movementCompositeType MovementCompositeType // Inherited from movement of parent mesoscopic link (if firstMovement = true)
	controlType           ControlType           // Inherited from mesoscopic link
	laneID                int                   // Inherited from source node
	isFirstMovement       bool                  // In the link is of movement type and it is the first microscopic link of the parent mesoscopic link
}
