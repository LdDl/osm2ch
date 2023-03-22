package osm2ch

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type NetworkMicroscopic struct {
	nodes map[NetworkNodeID]*NetworkNodeMicroscopic
	links map[NetworkLinkID]*NetworkLinkMicroscopic
	// Track ID generators
	maxLinkID NetworkLinkID
	maxNodeID NetworkNodeID
}

func (net *NetworkMicroscopic) ExportToCSV(fname string) error {

	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_micro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_micro_links.csv")

	err := net.exportNodesToCSV(fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export nodes")
	}

	err = net.exportLinksToCSV(fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export links")
	}
	return nil
}

func (net *NetworkMicroscopic) exportLinksToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, link := range net.links {
		_ = link
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}

func (net *NetworkMicroscopic) exportNodesToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "zone_id", "meso_link_id", "lane_number", "boundary_type", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, node := range net.nodes {
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.zoneID),
			fmt.Sprintf("%d", node.mesoLinkID),
			fmt.Sprintf("%d", node.laneID),
			fmt.Sprintf("%s", node.boundaryType),
			fmt.Sprintf("%f", node.geom[0]),
			fmt.Sprintf("%f", node.geom[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}
