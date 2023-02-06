package osm2ch

type NetworkMicroscopic struct {
	nodes map[NetworkNodeID]*NetworkNodeMicroscopic
	links map[NetworkLinkID]*NetworkLinkMicroscopic
	// Track ID generators
	maxLinkID NetworkLinkID
	maxNodeID NetworkNodeID
}
