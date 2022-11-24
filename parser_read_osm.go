package osm2ch

import "github.com/pkg/errors"

func (parser *Parser) createNetwork(verbose bool) (*NetworkMacroscopic, error) {
	/* Fill fields in case they haven't been provided earlier */
	if len(parser.networkTypes) == 0 {
		parser.networkTypes = []string{"auto"}
	}
	dataOSM, err := readOSM(parser.filename, verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't parse OSM data")
	}
	net, err := dataOSM.prepareNetwork(verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare road network")
	}
	return net, nil
}
