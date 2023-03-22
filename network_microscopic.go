package osm2ch

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/paulmach/orb/encoding/wkt"
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

	err = writer.Write([]string{"id", "source_node", "target_node", "meso_link_id", "macro_link_id", "macro_node_id", "cell_type", "lane_number", "meso_link_type", "control_type", "movement_composite_type", "free_speed", "capacity", "additional_travel_cost", "allowed_agent_types", "length_meters", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, link := range net.links {
		allowedAgentTypes := make([]string, len(link.allowedAgentTypes))
		for i, agentType := range link.allowedAgentTypes {
			allowedAgentTypes[i] = fmt.Sprintf("%s", agentType)
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", link.ID),
			fmt.Sprintf("%d", link.sourceNodeID),
			fmt.Sprintf("%d", link.targetNodeID),
			fmt.Sprintf("%d", link.mesoLinkID),
			fmt.Sprintf("%d", link.macroLinkID),
			fmt.Sprintf("%d", link.macroNodeID),
			fmt.Sprintf("%s", link.cellType),
			fmt.Sprintf("%d", link.laneID),
			fmt.Sprintf("%d", link.mesoLinkType),
			fmt.Sprintf("%s", link.controlType),
			fmt.Sprintf("%s", link.movementCompositeType),
			fmt.Sprintf("%f", link.freeSpeed),
			fmt.Sprintf("%d", link.capacity),
			fmt.Sprintf("%f", link.additionalTravelCost),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%f", link.lengthMeters),
			fmt.Sprintf("%s", wkt.MarshalString(link.geom)),
		})
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
