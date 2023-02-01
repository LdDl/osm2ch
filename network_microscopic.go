package osm2ch

type NetworkMicroscopic struct {
	nodes map[NetworkNodeID]*NetworkNodeMicroscopic
	links map[NetworkLinkID]*NetworkLinkMicroscopic
	// Track ID generator
	maxLinkID NetworkLinkID
}
