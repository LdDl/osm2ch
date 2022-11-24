package osm2ch

import (
	"github.com/pkg/errors"
)

func (data *OSMDataRaw) prepareWaysAndNodes(verbose bool) error {
	err := data.prepareWays(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't prepare ways")
	}
	err = data.prepareNodes(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't prepare nodes")
	}
	return nil
}
