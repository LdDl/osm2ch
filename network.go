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
	links map[NetworkLinkID]*NetworkLink
	nodes map[NetworkNodeID]*NetworkNode
}

func (net *NetworkMacroscopic) ExportToCSV(fname string) error {

	fnameParts := strings.Split(fname, ".csv")
	fnameNodes := fmt.Sprintf(fnameParts[0] + "_macro_nodes.csv")
	fnameLinks := fmt.Sprintf(fnameParts[0] + "_macro_links.csv")

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
			fmt.Sprintf("%d", link.lanesList[0]),
			fmt.Sprintf("%f", link.maxSpeed),
			fmt.Sprintf("%f", link.freeSpeed),
			fmt.Sprintf("%d", link.capacity),
			fmt.Sprintf("%f", link.lengthMeters),
			link.name,
			fmt.Sprintf("%s", wkt.Marshal(link.geom)),
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

func (net *NetworkMacroscopic) genActivityType() error {
	nodesLinkTypesCounters := make(map[NetworkNodeID]map[LinkType]int)
	for _, link := range net.links {
		sourceNodeID := link.sourceNodeID
		if _, ok := net.nodes[sourceNodeID]; !ok {
			return fmt.Errorf("No source node with ID '%d'. Link ID: '%d'", sourceNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID]; !ok {
			nodesLinkTypesCounters[sourceNodeID] = make(map[LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[sourceNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[sourceNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[sourceNodeID][link.linkType]++
		}

		targetNodeID := link.targetNodeID
		if _, ok := net.nodes[targetNodeID]; !ok {
			return fmt.Errorf("No target node with ID '%d'. Link ID: '%d'", targetNodeID, link.ID)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID]; !ok {
			nodesLinkTypesCounters[targetNodeID] = make(map[LinkType]int)
		}
		if _, ok := nodesLinkTypesCounters[targetNodeID][link.linkType]; !ok {
			nodesLinkTypesCounters[targetNodeID][link.linkType] = 1
		} else {
			nodesLinkTypesCounters[targetNodeID][link.linkType]++
		}
	}

	for nodeID, node := range net.nodes {
		if node.poiID > -1 {
			node.activityType = ACTIVITY_POI
			node.activityLinkType = LINK_UNDEFINED
		}
		if linkTypesCounters, ok := nodesLinkTypesCounters[nodeID]; ok {
			maxLinkType := LINK_UNDEFINED
			maxLinkTypeCount := 0
			for linkType, counter := range linkTypesCounters {
				if counter > maxLinkTypeCount {
					maxLinkTypeCount = counter
					maxLinkType = linkType
				}
			}
			if maxLinkType > 0 {
				node.activityType = ACTIVITY_LINK
				node.activityLinkType = maxLinkType
			} else {
				node.activityType = ACTIVITY_NONE
				node.activityLinkType = LINK_UNDEFINED
			}
		}
	}

	for _, node := range net.nodes {
		node.boundaryType = BOUNDARY_NONE
		if node.activityType == ACTIVITY_POI {
			continue
		}
		if len(node.outcomingLinks) == 0 {
			node.boundaryType = BOUNDARY_INCOME_ONLY
		} else if len(node.incomingLinks) == 0 {
			node.boundaryType = BOUNDARY_OUTCOME_ONLY
		} else if len(node.incomingLinks) == 1 && (len(node.outcomingLinks) == 1) {
			incomingLink, ok := net.links[node.incomingLinks[0]]
			if !ok {
				return fmt.Errorf("No incoming link with ID '%d'. Node ID: '%d'", node.incomingLinks[0], node.ID)
			}
			outcomingLink, ok := net.links[node.outcomingLinks[0]]
			if !ok {
				return fmt.Errorf("No incoming link with ID '%d'. Node ID: '%d'", node.outcomingLinks[0], node.ID)
			}
			if incomingLink.sourceNodeID == outcomingLink.targetNodeID {
				node.boundaryType = BOUNDARY_INCOME_OUTCOME
			}
		}
	}
	for _, node := range net.nodes {
		if node.boundaryType == BOUNDARY_NONE {
			continue
		}
		node.zoneID = node.ID
	}
	return nil
}
