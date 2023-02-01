package osm2ch

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

type NetworkMesoscopic struct {
	nodes map[NetworkNodeID]*NetworkNodeMesoscopic
	links map[NetworkLinkID]*NetworkLinkMesoscopic
	// Storage to track number of generated mesoscopic nodes for each macroscopic node which is centroid
	// Key: NodeID, Value: Number of expanded nodes
	expandedMesoNodes map[NetworkNodeID]int
	// Track ID generator
	maxLinkID NetworkLinkID
}

func (net *NetworkMesoscopic) ExportToCSV(fname string) error {

	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_meso_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_meso_links.csv")

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

func (net *NetworkMesoscopic) exportLinksToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "source_node", "target_node", "macro_node_id", "macro_link_id", "link_type", "control_type", "movement_composite_type", "allowed_agent_types", "lanes", "free_speed", "capacity", "length_meters", "geom"})
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
			fmt.Sprintf("%d", link.macroNodeID),
			fmt.Sprintf("%d", link.macroLinkID),
			fmt.Sprintf("%s", link.linkType),
			fmt.Sprintf("%s", link.controlType),
			fmt.Sprintf("%s", link.movementCompositeType),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%d", link.lanesNum),
			fmt.Sprintf("%f", link.freeSpeed),
			fmt.Sprintf("%d", link.capacity),
			fmt.Sprintf("%f", link.lengthMeters),
			fmt.Sprintf("%s", wkt.MarshalString(link.geom)),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}

func (net *NetworkMesoscopic) exportNodesToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "zone_id", "macro_node_id", "macro_link_id", "activity_link_type", "boundary_type", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, node := range net.nodes {
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.zoneID),
			fmt.Sprintf("%d", node.macroNodeID),
			fmt.Sprintf("%d", node.macroLinkID),
			fmt.Sprintf("%s", node.activityLinkType),
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
