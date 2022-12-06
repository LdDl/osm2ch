package osm2ch

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/paulmach/orb/encoding/wkt"
	"github.com/pkg/errors"
)

type NetworkMacroscopic struct {
	links    map[NetworkLinkID]*NetworkLink
	nodes    map[NetworkNodeID]*NetworkNode
	movement map[MovementID]*Movement
}

func (net *NetworkMacroscopic) ExportToCSV(fname string) error {

	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_macro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_macro_links.csv")
	fnameMovement := fmt.Sprintf(fnameParts[0] + "_movement.csv")

	err := net.exportNodesToCSV(fnameNodes)
	if err != nil {
		return errors.Wrap(err, "Can't export nodes")
	}

	err = net.exportLinksToCSV(fnameLinks)
	if err != nil {
		return errors.Wrap(err, "Can't export links")
	}

	err = net.exportMovementToCSV(fnameMovement)
	if err != nil {
		return errors.Wrap(err, "Can't export movement")
	}

	return nil
}

func (net *NetworkMacroscopic) exportLinksToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "source_node", "target_node", "osm_way_id", "link_class", "is_link", "link_type", "control_type", "allowed_agent_types", "was_bidirectional", "lanes", "max_speed", "free_speed", "capacity", "length_meters", "name", "geom"})
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
			fmt.Sprintf("%d", link.osmWayID),
			fmt.Sprintf("%s", link.linkClass),
			fmt.Sprintf("%s", link.linkConnectionType),
			fmt.Sprintf("%s", link.linkType),
			fmt.Sprintf("%s", link.controlType),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%t", link.wasBidirectional),
			fmt.Sprintf("%d", link.GetLanes()),
			fmt.Sprintf("%f", link.maxSpeed),
			fmt.Sprintf("%f", link.freeSpeed),
			fmt.Sprintf("%d", link.capacity),
			fmt.Sprintf("%f", link.lengthMeters),
			link.name,
			fmt.Sprintf("%s", wkt.MarshalString(link.geom)),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write link")
		}
	}
	return nil
}

func (net *NetworkMacroscopic) exportNodesToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "osm_node_id", "control_type", "boundary_type", "activity_type", "activity_link_type", "zone_id", "intersection_id", "poi_id", "osm_highway", "name", "longitude", "latitude"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, node := range net.nodes {
		err = writer.Write([]string{
			fmt.Sprintf("%d", node.ID),
			fmt.Sprintf("%d", node.osmNodeID),
			fmt.Sprintf("%s", node.controlType),
			fmt.Sprintf("%s", node.boundaryType),
			fmt.Sprintf("%s", node.activityType),
			fmt.Sprintf("%s", node.activityLinkType),
			fmt.Sprintf("%d", node.zoneID),
			fmt.Sprintf("%d", node.intersectionID),
			fmt.Sprintf("%d", node.poiID),
			node.osmHighway,
			node.name,
			fmt.Sprintf("%f", node.geom[0]),
			fmt.Sprintf("%f", node.geom[1]),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write node")
		}
	}
	return nil
}

func (net *NetworkMacroscopic) exportMovementToCSV(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return errors.Wrap(err, "Can't create file")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	err = writer.Write([]string{"id", "node_id", "osm_node_id", "name", "in_link_id", "in_lane_start", "in_lane_end", "out_link_id", "out_lane_start", "out_lane_end", "lanes_num", "from_osm_node_id", "to_osm_node_id", "type", "penalty", "capacity", "control_type", "movement_composite_type", "volume", "free_speed", "allowed_agent_types", "geom"})
	if err != nil {
		return errors.Wrap(err, "Can't write header")
	}

	for _, mvmt := range net.movement {
		allowedAgentTypes := make([]string, len(mvmt.allowedAgentTypes))
		for i, agentType := range mvmt.allowedAgentTypes {
			allowedAgentTypes[i] = fmt.Sprintf("%s", agentType)
		}
		err = writer.Write([]string{
			fmt.Sprintf("%d", mvmt.ID),
			fmt.Sprintf("%d", mvmt.NodeID),
			fmt.Sprintf("%d", mvmt.osmNodeID),
			fmt.Sprintf("%s", "-"),
			fmt.Sprintf("%d", mvmt.IncomingLinkID),
			fmt.Sprintf("%d", mvmt.incomeLaneStart),
			fmt.Sprintf("%d", mvmt.incomeLaneEnd),
			fmt.Sprintf("%d", mvmt.OutcomingLinkID),
			fmt.Sprintf("%d", mvmt.outcomeLaneStart),
			fmt.Sprintf("%d", mvmt.outcomeLaneEnd),
			fmt.Sprintf("%d", mvmt.lanesNum),
			fmt.Sprintf("%d", mvmt.fromOsmNodeID),
			fmt.Sprintf("%d", mvmt.toOsmNodeID),
			fmt.Sprintf("%s", mvmt.movementType),
			fmt.Sprintf("%d", -1),
			fmt.Sprintf("%d", -1),
			fmt.Sprintf("%s", mvmt.controlType),
			fmt.Sprintf("%s", mvmt.movementCompositeType),
			fmt.Sprintf("%d", -1),
			fmt.Sprintf("%d", -1),
			strings.Join(allowedAgentTypes, ","),
			fmt.Sprintf("%s", wkt.MarshalString(mvmt.geom)),
		})
		if err != nil {
			return errors.Wrap(err, "Can't write movement")
		}
	}
	return nil
}
