package osm2ch

import "github.com/paulmach/orb"

type NetworkNodeMicroscopic struct {
	geom orb.Point

	ID NetworkNodeID

	laneID                     int
	isLinkUpstreamTargetNode   bool
	isLinkDownstreamTargetNode bool
	zoneID                     NetworkNodeID // Should be inherited from the macroscopic node which is target (isLinkUpstreamTargetNode = true) or source (isLinkDownstreamTargetNode = true) node of parent mesoscopic link
	boundaryType               BoundaryType  // Should be evaluated from macroscopic node which is target (isLinkUpstreamTargetNode = true) or source (isLinkDownstreamTargetNode = true) node of parent mesoscopic link
}
