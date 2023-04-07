package osm2ch

import "github.com/pkg/errors"

func (parser *Parser) createNetwork(verbose bool, poi bool) (*NetworkMacroscopic, error) {
	/* Fill fields in case they haven't been provided earlier */
	if len(parser.networkTypes) == 0 {
		parser.networkTypes = []string{"auto"}
	}
	dataOSM, err := readOSM(parser.filename, verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't parse OSM data")
	}
	net, err := dataOSM.prepareNetwork(verbose, poi)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare road network")
	}

	// @TODO: Postprocess isolated nodes
	// @TODO: Postproces adjacent links at 2-degree nodes
	// @TODO: Postprocess two-way overlapping links (by offsetting the geometry)
	return net, nil
}
