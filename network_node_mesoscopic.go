package osm2ch

import (
	"github.com/paulmach/orb"
)

type NetworkNodeMesoscopic struct {
	incomingLinks  []NetworkLinkID
	outcomingLinks []NetworkLinkID

	geom          orb.Point
	geomEuclidean orb.Point

	ID          NetworkNodeID
	macroNodeID NetworkNodeID
	macroLinkID NetworkLinkID

	zoneID           NetworkNodeID // Should be inherited from the macroscopic node
	activityLinkType LinkType      // Should be inherited from the macroscopic node
	boundaryType     BoundaryType  // Should be evaluated from macroscopic node and macroscopic link
}
