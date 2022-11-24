package osm2ch

import "github.com/pkg/errors"

func (parser *Parser) createNetwork(verbose bool) error {
	/* Fill fields in case they haven't been provided earlier */
	if len(parser.networkTypes) == 0 {
		parser.networkTypes = []string{"auto"}
	}

	dataOSM, err := readOSM(parser.filename, verbose)
	if err != nil {
		return errors.Wrap(err, "Can't parse OSM data")
	}
	err = dataOSM.prepareNetwork(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't prepare data")
	}
	return nil
}
