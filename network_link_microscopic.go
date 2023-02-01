package osm2ch

import (
	"github.com/paulmach/orb"
)

type MicroscopicLinkType uint16

const (
	LINK_FORWARD = MicroscopicLinkType(1)
	LINK_LANE_CHANGE
)

type NetworkLinkMicroscopic struct {
	geom          orb.LineString
	geomEuclidean orb.LineString

	ID NetworkLinkID

	sourceNodeID NetworkNodeID // Corresponds to ID of microscopic node (not to meso or macro or OSM)
	targetNodeID NetworkNodeID // Corresponds to ID of microscopic node (not to meso or macro or OSM)

	mesoLinkID NetworkLinkID

	microLinkType         MicroscopicLinkType
	allowedAgentTypes     []AgentType
	firstMovement         bool
	movementCompositeType MovementCompositeType // Inherited from movement of parent mesoscopic link (if firstMovement = true)
	controlType           ControlType           // Inherited from mesoscipoc link
	laneID                int                   // Inherited from source node
}
