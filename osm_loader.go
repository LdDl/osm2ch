package osm2ch

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	geojson "github.com/paulmach/go.geojson"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"

	"github.com/paulmach/osm/osmpbf"
)

const (
	earthRadius = 6370.986884258304
	pi180       = math.Pi / 180.0
	pi180Rev    = 180.0 / math.Pi
)

// GeoPoint Representation of point on Earth
type GeoPoint struct {
	Lat float64
	Lon float64
}

// String Pretty printing for GeoPoint
func (gp GeoPoint) String() string {
	return fmt.Sprintf("Lon: %f | Lat: %f", gp.Lon, gp.Lat)
}

// edgeComponent Representation of edge (vertex_from -> vertex_to)
type edgeComponent struct {
	from int64
	to   int64
}

// wayComponent First and last edges of osm.Way
type wayComponent struct {
	FirstEdge edgeComponent
	LastEdge  edgeComponent
}

// restrictionComponent Representation of member of restriction relation. Could be way or node.
type restrictionComponent struct {
	ID   int64
	Type string
}

// expandedEdge New edge built on top of two adjacent edges
type expandedEdge struct {
	ID        int64
	Cost      float64
	Geom      []GeoPoint
	WasOneWay bool // Former OSM object was one way.
}

// ExpandedGraph Representation of edge expanded graph
/*
	map[newSourceVertexID]map[newTargetVertexID]newExpandedEdge
*/
type ExpandedGraph map[int64]map[int64]expandedEdge

type Edge struct {
	ID        int64
	WayID     osm.WayID
	Source    osm.NodeID
	Target    osm.NodeID
	WasOneway bool
	Geom      []GeoPoint
}

// ImportFromOSMFile Imports graph from file of PBF-format (in OSM terms)
/*
	File should have PBF (Protocolbuffer Binary Format) extension according to https://github.com/paulmach/osm
*/
func ImportFromOSMFile(fileName string, cfg *OsmConfiguration) (ExpandedGraph, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "File open")
	}
	defer f.Close()

	scannerWays := osmpbf.New(context.Background(), f, 4)
	defer scannerWays.Close()

	ways := []Way{}
	nodes := make(map[osm.NodeID]Node)
	nodesSeen := make(map[osm.NodeID]struct{})

	fmt.Printf("Scanning ways...")
	st := time.Now()
	for scannerWays.Scan() {
		obj := scannerWays.Object()
		if obj.ObjectID().Type() != "way" {
			continue
		}
		way := obj.(*osm.Way)
		tagMap := way.TagMap()
		tag, ok := tagMap[cfg.EntityName]
		if !ok {
			continue
		}
		if !cfg.CheckTag(tag) {
			continue
		}
		oneway := false
		if v, ok := tagMap["oneway"]; ok {
			if v == "yes" || v == "1" {
				oneway = true
			}
		}
		nodes := way.Nodes
		preparedWay := Way{
			ID:     way.ID,
			Nodes:  make(osm.WayNodes, len(nodes)),
			Oneway: oneway,
			TagMap: make(osm.Tags, len(way.Tags)),
		}
		copy(preparedWay.Nodes, nodes)
		copy(preparedWay.TagMap, way.Tags)
		ways = append(ways, preparedWay)
		for _, node := range nodes {
			nodesSeen[node.ID] = struct{}{}
		}
	}
	if scannerWays.Err() != nil {
		return nil, errors.Wrap(scannerWays.Err(), "Scanner error on Ways")
	}
	fmt.Printf("Done in %v\n\tWays: %d\n", time.Since(st), len(ways))

	// Seek file to start
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after ways scanning")
	}
	scannerNodes := osmpbf.New(context.Background(), f, 4)
	defer scannerNodes.Close()

	fmt.Printf("Scanning nodes...")
	st = time.Now()
	for scannerNodes.Scan() {
		obj := scannerNodes.Object()
		if obj.ObjectID().Type() != "node" {
			continue
		}
		node := obj.(*osm.Node)
		if _, ok := nodesSeen[node.ID]; ok {
			delete(nodesSeen, node.ID)
			nodes[node.ID] = Node{
				ID:       node.ID,
				useCount: 0,
				node:     *node,
			}
		}
	}
	if scannerNodes.Err() != nil {
		return nil, errors.Wrap(scannerNodes.Err(), "Scanner error on Nodes")
	}
	fmt.Printf("Done in %v\n\tNodes: %d\n", time.Since(st), len(nodes))

	// @todo: scan maneuvers (restrictions)
	// Seek file to start
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after nodes scanning")
	}
	scannerManeuvers := osmpbf.New(context.Background(), f, 4)
	defer scannerManeuvers.Close()
	fmt.Printf("Scanning maneuvers (restrictions)...")
	st = time.Now()
	skippedRestrictions := 0
	unsupportedRestrictionRoles := 0
	possibleRestrictionCombos := make(map[string]map[string]bool)
	restrictions := make(map[string]map[restrictionComponent]map[restrictionComponent]restrictionComponent)
	for scannerNodes.Scan() {
		obj := scannerNodes.Object()
		if obj.ObjectID().Type() == "relation" {
			relation := obj.(*osm.Relation)
			tagMap := relation.TagMap()
			tag, ok := tagMap["restriction"]
			if !ok {
				continue
			}
			_ = tag
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
	}
	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tSkipped restrictions (which have not exactly 3 members): %d\n", skippedRestrictions)
	fmt.Printf("\tNumber of unknow restriction roles (only 'from', 'to' and 'via' supported): %d\n", unsupportedRestrictionRoles)

	fmt.Printf("Counting node use cases...")
	st = time.Now()
	for _, way := range ways {
		for i, wayNode := range way.Nodes {
			if node, ok := nodes[wayNode.ID]; ok {
				if i == 0 || i == len(way.Nodes)-1 {
					node.useCount += 2
					nodes[wayNode.ID] = node
				} else {
					node.useCount += 1
					nodes[wayNode.ID] = node
				}
			} else {
				return nil, fmt.Errorf("Missing node with id: %d\n", wayNode.ID)
			}
		}
	}
	fmt.Printf("Done in %v\n", time.Since(st))

	fmt.Printf("Preparing edges...")
	st = time.Now()
	edges := []Edge{}
	onewayEdges := 0
	totalEdgesNum := int64(0)
	for _, way := range ways {
		var source osm.NodeID
		geometry := []GeoPoint{}
		for i, wayNode := range way.Nodes {
			node := nodes[wayNode.ID]
			if i == 0 {
				source = wayNode.ID
				geometry = append(geometry, GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat})
			} else {
				geometry = append(geometry, GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat})
				if node.useCount > 1 {
					totalEdgesNum++
					onewayEdges++
					if !way.Oneway {
						totalEdgesNum++
						edges = append(edges, Edge{
							ID:        totalEdgesNum,
							WayID:     way.ID,
							Source:    wayNode.ID,
							Target:    source,
							Geom:      reverseLine(geometry),
							WasOneway: false,
						})
					} else {
						edges = append(edges, Edge{
							ID:        totalEdgesNum,
							WayID:     way.ID,
							Source:    source,
							Target:    wayNode.ID,
							Geom:      geometry,
							WasOneway: true,
						})
					}
					source = wayNode.ID
					geometry = []GeoPoint{GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat}}
				}
			}
		}
	}
	fmt.Printf("Done in %v\n\tEdges: (oneway = %d), (not oneway = %d)\n", time.Since(st), onewayEdges, totalEdgesNum)

	fmt.Printf("Preparing nodes...")
	st = time.Now()
	nodesFiltered := []Node{}
	for _, node := range nodes {
		if node.useCount > 1 {
			nodesFiltered = append(nodesFiltered, node)
		}
	}
	fmt.Printf("Done in %v\n\tNodes: %d\n", time.Since(st), len(nodesFiltered))

	// @todo: work with maneuvers (restrictions)
	fmt.Printf("Working with maneuvers (restrictions)...")
	st = time.Now()
	immposibleRestrictions := 0
	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tNot properly handeled restrictions: %d\n", immposibleRestrictions)

	// @todo: expand
	fmt.Println("Applying edge expanding technique...")
	expandedGraph := make(ExpandedGraph)

	return expandedGraph, nil
}

// degreesToRadians deg = r * pi / 180
func degreesToRadians(d float64) float64 {
	return d * pi180
}

// radiansTodegrees r = deg  * 180 / pi
func radiansTodegrees(d float64) float64 {
	return d * pi180Rev
}

// greatCircleDistance Returns distance between two geo-points (kilometers)
func greatCircleDistance(p, q GeoPoint) float64 {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)
	diffLat := lat2 - lat1
	diffLon := lon2 - lon1
	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	ans := c * earthRadius
	return ans
}

func middlePoint(p, q GeoPoint) GeoPoint {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)

	Bx := math.Cos(lat2) * math.Cos(lon2-lon1)
	By := math.Cos(lat2) * math.Sin(lon2-lon1)

	latMid := math.Atan2(math.Sin(lat1)+math.Sin(lat2), math.Sqrt((math.Cos(lat1)+Bx)*(math.Cos(lat1)+Bx)+By*By))
	lonMid := lon1 + math.Atan2(By, math.Cos(lat1)+Bx)
	return GeoPoint{Lat: radiansTodegrees(latMid), Lon: radiansTodegrees(lonMid)}
}

func reverseLine(pts []GeoPoint) []GeoPoint {
	inputLen := len(pts)
	output := make([]GeoPoint, inputLen)
	for i, n := range pts {
		j := inputLen - i - 1
		output[j] = n
	}
	return output
}

func reverseLineInPlace(pts []GeoPoint) {
	inputLen := len(pts)
	inputMid := inputLen / 2
	for i := 0; i < inputMid; i++ {
		j := inputLen - i - 1
		pts[i], pts[j] = pts[j], pts[i]
	}
}

// PrepareWKTLinestring Creates WKT LineString from set of points
func PrepareWKTLinestring(pts []GeoPoint) string {
	ptsStr := make([]string, len(pts))
	for i := range pts {
		ptsStr[i] = fmt.Sprintf("%f %f", pts[i].Lon, pts[i].Lat)
	}
	return fmt.Sprintf("LINESTRING(%s)", strings.Join(ptsStr, ","))
}

// PrepareGeoJSONLinestring Creates GeoJSON LineString from set of points
func PrepareGeoJSONLinestring(pts []GeoPoint) string {
	pts2d := make([][]float64, len(pts))
	for i := range pts {
		pts2d[i] = []float64{pts[i].Lon, pts[i].Lat}
	}
	b, err := geojson.NewLineStringGeometry(pts2d).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}

// PrepareWKTPoint Creates WKT Point from given points
func PrepareWKTPoint(pt GeoPoint) string {
	return fmt.Sprintf("POINT(%f %f)", pt.Lon, pt.Lat)
}

// PrepareGeoJSONPoint Creates GeoJSON Point from given point
func PrepareGeoJSONPoint(pt GeoPoint) string {
	b, err := geojson.NewPointGeometry([]float64{pt.Lon, pt.Lat}).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}
