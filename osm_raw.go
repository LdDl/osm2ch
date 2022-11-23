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
	waysRaw      []*WayData
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
			preparedWay.flattenTags(verbose)
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
		waysRaw:      ways,
		nodes:        nodes,
		restrictions: restrictions,
	}
	if len(data.allowedAgentTypes) == 0 {
		data.allowedAgentTypes = []AgentType{AGENT_AUTO}
	}
	return &data, nil
}

func (data *OSMDataRaw) prepare(verbose bool) {
	data.prepareMedium(verbose)
	data.prepareWellDone(verbose)

	for _, way := range data.waysMedium {
		fmt.Println(way.ID, way.linkClass, way.linkType, way.linkConnectionType, way.Oneway, way.lanes, way.maxSpeed)
	}
}
