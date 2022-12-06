package osm2ch

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
	"github.com/pkg/errors"
)

type OSMScanner interface {
	Scan() bool
	Close() error
	Err() error
	Object() osm.Object
}

type OSMDataRaw struct {
	restrictions map[string]map[restrictionComponent]map[restrictionComponent]restrictionComponent
	nodes        map[osm.NodeID]*Node
	ways         []*WayData
	waysMedium   []*WayData

	allowedAgentTypes []AgentType
}

func readOSM(filename string, verbose bool) (*OSMDataRaw, error) {
	if verbose {
		fmt.Printf("Opening file: '%s'...\n", filename)
	}
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	/* Process ways */
	if verbose {
		fmt.Printf("\tProcessing ways... ")
	}
	st := time.Now()
	ways := []*WayData{}
	nodesSeen := make(map[osm.NodeID]struct{})
	{
		var scannerWays OSMScanner

		// Guess file extension and prepare correct scanner for ways
		ext := filepath.Ext(filename)
		switch ext {
		case ".osm", ".xml":
			scannerWays = osmxml.New(context.Background(), file)
		case ".pbf", ".osm.pbf":
			scannerWays = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("File extension '%s' for file '%s' is not handled yet", ext, filename)
		}
		defer scannerWays.Close()

		// Scan ways
		for scannerWays.Scan() {
			obj := scannerWays.Object()
			if obj.ObjectID().Type() != "way" {
				continue
			}
			way := obj.(*osm.Way)
			oneway := false
			onewayDefault := false
			isReversed := false
			onewayText := way.Tags.Find("oneway")
			if onewayText != "" {
				if onewayText == "yes" || onewayText == "1" {
					oneway = true
				} else if onewayText == "no" || onewayText == "0" {
					oneway = false
				} else if onewayText == "-1" {
					oneway = true
					isReversed = true
				} else {
					// Reversible or alternating
					// Those are depends on time conditions
					// @todo: need to implement
					if _, found := onewayReversible[onewayText]; found {
						oneway = false
					} else {
						fmt.Printf("[WARNING]: Unhandled `oneway` tag value has been met: '%s'. Way ID: '%d'", onewayText, way.ID)
					}
				}
			} else {
				junctionText := way.Tags.Find("junction")
				if _, ok := junctionTypes[junctionText]; ok {
					oneway = true
				} else {
					oneway = false
					onewayDefault = true
				}
			}
			preparedWay := &WayData{
				ID:            way.ID,
				Oneway:        oneway,
				OnewayDefault: onewayDefault,
				IsReversed:    isReversed,
				Nodes:         make([]osm.NodeID, 0, len(way.Nodes)),
				TagMap:        make(osm.Tags, len(way.Tags)),

				maxSpeed:      -1.0,
				freeSpeed:     -1.0,
				capacity:      -1.0,
				lanes:         -1,
				lanesForward:  -1,
				lanesBackward: -1,
			}
			copy(preparedWay.TagMap, way.Tags)
			// Mark way's nodes as seen to remove isolated nodes in further
			for _, node := range way.Nodes {
				nodesSeen[node.ID] = struct{}{}
				preparedWay.Nodes = append(preparedWay.Nodes, node.ID)
			}
			// Call tags flattening to make further processing easier
			preparedWay.processTags(verbose)
			ways = append(ways, preparedWay)
		}
		err = scannerWays.Err()
		if err != nil {
			return nil, err
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	// Seek file to start
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after ways scanning")
	}

	/* Process nodes */
	if verbose {
		fmt.Printf("\tProcessing nodes... ")
	}
	st = time.Now()
	nodes := make(map[osm.NodeID]*Node)
	{

		var scannerNodes OSMScanner

		// Guess file extension and prepare correct scanner for ways
		ext := filepath.Ext(filename)
		switch ext {
		case ".osm", ".xml":
			scannerNodes = osmxml.New(context.Background(), file)
		case ".pbf", ".osm.pbf":
			scannerNodes = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("File extension '%s' for file '%s' is not handled yet", ext, filename)
		}
		defer scannerNodes.Close()

		// Scan nodes
		for scannerNodes.Scan() {
			obj := scannerNodes.Object()
			if obj.ObjectID().Type() != "node" {
				continue
			}
			node := obj.(*osm.Node)
			if _, ok := nodesSeen[node.ID]; ok {
				delete(nodesSeen, node.ID)
				nameText := node.Tags.Find("name")
				highwayText := node.Tags.Find("highway")
				controlType := NOT_SIGNAL
				if highwayText == "traffic_signals" {
					controlType = IS_SIGNAL
				}
				nodes[node.ID] = &Node{
					name:        nameText,
					node:        *node,
					ID:          node.ID,
					useCount:    0,
					isCrossing:  false,
					controlType: controlType,
					highway:     highwayText,
				}
			}
		}
		err = scannerNodes.Err()
		if err != nil {
			return nil, err
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	// Seek file to start
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after nodes scanning")
	}

	/* Process maneuvers (turn restrictions only)*/
	if verbose {
		fmt.Printf("\tProcessing maneuvers... ")
	}
	st = time.Now()
	skippedRestrictions := 0
	unsupportedRestrictionRoles := 0
	possibleRestrictionCombos := make(map[string]map[string]bool)
	restrictions := make(map[string]map[restrictionComponent]map[restrictionComponent]restrictionComponent)
	{
		var scannerRelations OSMScanner

		// Guess file extension and prepare correct scanner for ways
		ext := filepath.Ext(filename)
		switch ext {
		case ".osm", ".xml":
			scannerRelations = osmxml.New(context.Background(), file)
		case ".pbf", ".osm.pbf":
			scannerRelations = osmpbf.New(context.Background(), file, 4)
		default:
			return nil, fmt.Errorf("File extension '%s' for file '%s' is not handled yet", ext, filename)
		}
		defer scannerRelations.Close()

		// Scan relations
		for scannerRelations.Scan() {
			obj := scannerRelations.Object()
			if obj.ObjectID().Type() != "relation" {
				continue
			}
			relation := obj.(*osm.Relation)
			tagMap := relation.TagMap()
			tag, ok := tagMap["restriction"]
			if !ok {
				// Ignore non-restriction relations
				continue
			}
			members := relation.Members
			if len(members) != 3 {
				skippedRestrictions++
				// fmt.Printf("Restriction does not contain 3 members, relation ID: %d. Skip it\n", relation.ID)
				continue
			}
			firstMember := restrictionComponent{-1, ""}
			secondMember := restrictionComponent{-1, ""}
			thirdMember := restrictionComponent{-1, ""}

			switch members[0].Role {
			case "from":
				firstMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for first member of relation with ID: %d\n", relation.ID)
				break
			}

			switch members[1].Role {
			case "from":
				firstMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for second member of relation with ID: %d\n", relation.ID)
				break
			}

			switch members[2].Role {
			case "from":
				firstMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for third member of relation with ID: %d\n", relation.ID)
				break
			}
			if _, ok := possibleRestrictionCombos[tag]; !ok {
				possibleRestrictionCombos[tag] = make(map[string]bool)
			}
			possibleRestrictionCombos[tag][fmt.Sprintf("%s;%s;%s", firstMember.Type, secondMember.Type, thirdMember.Type)] = true

			if _, ok := restrictions[tag]; !ok {
				restrictions[tag] = make(map[restrictionComponent]map[restrictionComponent]restrictionComponent)
			}
			if _, ok := restrictions[tag][firstMember]; !ok {
				restrictions[tag][firstMember] = make(map[restrictionComponent]restrictionComponent)
			}
			if _, ok := restrictions[tag][firstMember][secondMember]; !ok {
				restrictions[tag][firstMember][secondMember] = thirdMember
			}
		}
		err = scannerRelations.Err()
		if err != nil {
			return nil, err
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}

	if verbose {
		fmt.Printf("Number of ways: %d\n", len(ways))
		fmt.Printf("Number of nodes: %d\n", len(nodes))
		fmt.Printf("Skipped restrictions (which have not exactly 3 members): %d\n", skippedRestrictions)
		fmt.Printf("Number of unknow restriction roles (only 'from', 'to' and 'via' supported): %d\n", unsupportedRestrictionRoles)
	}

	data := OSMDataRaw{
		ways:         ways,
		nodes:        nodes,
		restrictions: restrictions,
	}
	if len(data.allowedAgentTypes) == 0 {
		data.allowedAgentTypes = make([]AgentType, len(agentTypesDefault))
		copy(data.allowedAgentTypes, agentTypesDefault)
	}
	return &data, nil
}

func (data *OSMDataRaw) prepareNetwork(verbose bool) (*NetworkMacroscopic, error) {
	err := data.prepareWaysAndNodes(verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare ways or nodes")
	}
	err = data.markPureCycles(verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't mark pure cycles")
	}
	nodes, links, err := data.prepareNodesAndLinks(verbose)
	if err != nil {
		return nil, errors.Wrap(err, "Can't prepare links")
	}
	for _, link := range links {
		link.geomEuclidean = lineToEuclidean(link.geom)
	}
	for _, node := range nodes {
		node.geomEuclidean = pointToEuclidean(node.geom)
	}
	net := NetworkMacroscopic{
		nodes:    nodes,
		links:    links,
		movement: make(map[MovementID]*Movement),
	}
	return &net, nil
}

func (data *OSMDataRaw) prepareNodesAndLinks(verbose bool) (map[NetworkNodeID]*NetworkNode, map[NetworkLinkID]*NetworkLink, error) {
	lastLinkID := NetworkLinkID(0)
	lastNodeID := NetworkNodeID(0)

	observed := make(map[osm.NodeID]NetworkNodeID)
	nodes := make(map[NetworkNodeID]*NetworkNode)
	links := make(map[NetworkLinkID]*NetworkLink)

	for _, way := range data.waysMedium {
		if way.isPureCycle {
			continue
		}
		way.prepareSegments(data.nodes)
		for _, segment := range way.segments {
			if len(segment) < 2 {
				continue
			}
			var currentSourceNodeID NetworkNodeID
			var currentTargetNodeID NetworkNodeID
			/* Create nodes */
			sourceNodeID := segment[0]
			if nID, ok := observed[sourceNodeID]; !ok {
				sourceNode, ok := data.nodes[sourceNodeID]
				if !ok {
					return nil, nil, fmt.Errorf("No such source node '%d'. Way ID: '%d'", sourceNodeID, way.ID)
				}
				nodes[lastNodeID] = networkNodeFromOSM(lastNodeID, sourceNode)
				observed[sourceNodeID] = lastNodeID
				currentSourceNodeID = lastNodeID
				lastNodeID++
			} else {
				currentSourceNodeID = nID
			}
			targetNodeID := segment[len(segment)-1]
			if nID, ok := observed[targetNodeID]; !ok {
				targetNode, ok := data.nodes[targetNodeID]
				if !ok {
					return nil, nil, fmt.Errorf("No such target node '%d'. Way ID: '%d'", targetNodeID, way.ID)
				}
				nodes[lastNodeID] = networkNodeFromOSM(lastNodeID, targetNode)
				observed[targetNodeID] = lastNodeID
				currentTargetNodeID = lastNodeID
				lastNodeID++
			} else {
				currentTargetNodeID = nID
			}

			/* Create links */
			nodesForSegment := make([]*Node, len(segment))
			for i, nodeID := range segment {
				nodesForSegment[i] = data.nodes[nodeID]
			}
			links[lastLinkID] = networkLinkFromOSM(lastLinkID, currentSourceNodeID, currentTargetNodeID, nodes[currentSourceNodeID].osmNodeID, nodes[currentTargetNodeID].osmNodeID, DIRECTION_FORWARD, way, nodesForSegment)
			nodes[currentSourceNodeID].outcomingLinks = append(nodes[currentSourceNodeID].outcomingLinks, lastLinkID)
			nodes[currentTargetNodeID].incomingLinks = append(nodes[currentTargetNodeID].incomingLinks, lastLinkID)
			lastLinkID++
			if !way.Oneway {
				links[lastLinkID] = networkLinkFromOSM(lastLinkID, currentTargetNodeID, currentSourceNodeID, nodes[currentTargetNodeID].osmNodeID, nodes[currentSourceNodeID].osmNodeID, DIRECTION_BACKWARD, way, nodesForSegment)
				nodes[currentTargetNodeID].outcomingLinks = append(nodes[currentTargetNodeID].outcomingLinks, lastLinkID)
				nodes[currentSourceNodeID].incomingLinks = append(nodes[currentSourceNodeID].incomingLinks, lastLinkID)
				lastLinkID++
			}
		}
	}
	return nodes, links, nil
}

func (way *WayData) prepareSegments(nodes map[osm.NodeID]*Node) {
	nodesNum := len(way.Nodes)
	lastNodeIdx := 0
	idx := 0
	for {
		segmentNodes := []osm.NodeID{way.Nodes[lastNodeIdx]}
		for idx = lastNodeIdx + 1; idx < nodesNum; idx++ {
			nextNodeID := way.Nodes[idx]
			nextNode := nodes[nextNodeID]
			segmentNodes = append(segmentNodes, nextNodeID)
			if nextNode.isCrossing {
				lastNodeIdx = idx
				break
			}
		}
		way.segments = append(way.segments, segmentNodes)
		if idx == nodesNum-1 {
			break
		}
	}
}
