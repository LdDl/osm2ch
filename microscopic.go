package osm2ch

func genMicroscopicNetwork(macroNet *NetworkMacroscopic, mesoNet *NetworkMesoscopic, verbose bool) (*NetworkMicroscopic, error) {
	microscopic := NetworkMicroscopic{
		nodes: make(map[NetworkNodeID]*NetworkNodeMicroscopic),
		links: make(map[NetworkLinkID]*NetworkLinkMicroscopic),
	}

	// @TODO: Implement this function

	return &microscopic, nil
}
