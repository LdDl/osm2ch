package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LdDl/ch"
	"github.com/LdDl/osm2ch"
	"github.com/pkg/errors"
)

var (
	tagStr        = flag.String("tags", "motorway,primary,primary_link,road,secondary,secondary_link,residential,tertiary,tertiary_link,unclassified,trunk,trunk_link,motorway_link", "Set of needed tags (separated by commas)")
	osmFileName   = flag.String("file", "my_graph.osm.pbf", "Filename of *.osm.pbf file (it has to be compressed)")
	out           = flag.String("out", "my_graph.csv", "Filename of 'Comma-Separated Values' (CSV) formatted file. E.g.: if file name is 'map.csv' then 3 files will be produced: 'map.csv' (edges), 'map_vertices.csv', 'map_shortcuts.csv'")
	geomFormat    = flag.String("geomf", "wkt", "Format of output geometry. Expected values: wkt / geojson")
	units         = flag.String("units", "km", "Units of output weights. Expected values: km for kilometers / m for meters")
	doContraction = flag.Bool("contract", true, "Prepare contraction hierarchies?")
)

func main() {

	flag.Parse()

	tags := strings.Split(*tagStr, ",")
	cfg := osm2ch.OsmConfiguration{
		EntityName: "highway", // Currrently we do not support others
		Tags:       tags,
	}

	edgeExpandedGraph, err := osm2ch.ImportFromOSMFile(*osmFileName, &cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fnamePart := strings.Split(*out, ".csv") // to guarantee proper filename and its extension
	fnameEdges := fmt.Sprintf(fnamePart[0] + ".csv")
	fnameVertices := fmt.Sprintf(fnamePart[0] + "_vertices.csv")
	fnameShortcuts := fmt.Sprintf(fnamePart[0] + "_shortcuts.csv")
	/* Edges file */
	fileEdges, err := os.Create(fnameEdges)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fileEdges.Close()
	writerEdges := csv.NewWriter(fileEdges)
	defer writerEdges.Flush()
	writerEdges.Comma = ';'
	// 		from_vertex_id - int64, ID of generated source vertex
	// 		to_vertex_id - int64, ID of generated target vertex
	// 		weight - float64, Weight of an edge (meters/kilometers)
	//      geom - geometry (WKT or GeoJSON representation)
	//      was_one_way - if edge was one way
	//      edge_id - int64, ID of generated edge
	// 		osm_way_from - int64, ID of source OSM Way
	// 		osm_way_to - int64, ID of target OSM Way
	// 		osm_way_from_source_node - int64, ID of first OSM Node in source OSM Way
	// 		osm_way_from_target_node - int64, ID of last OSM Node in source OSM Way
	// 		osm_way_to_source_node - int64, ID of first OSM Node in target OSM Way
	// 		osm_way_to_target_node - int64, ID of last OSM Node in target OSM Way
	err = writerEdges.Write([]string{"from_vertex_id", "to_vertex_id", "weight", "geom", "was_one_way", "edge_id", "osm_way_from", "osm_way_to", "osm_way_from_source_node", "osm_way_from_target_node", "osm_way_to_source_node", "osm_way_to_target_node"})
	if err != nil {
		fmt.Println(err)
		return
	}

	/* Vertices file */
	fileVertices, err := os.Create(fnameVertices)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fileVertices.Close()
	writerVertices := csv.NewWriter(fileVertices)
	defer writerVertices.Flush()
	writerVertices.Comma = ';'
	// 		vertex_id - int64, ID of vertex
	// 		order_pos - int, Position of vertex in hierarchies (evaluted by library)
	// 		importance - int, Importance of vertex in graph (evaluted by library)
	//      geom - geometry (WKT or GeoJSON representation)
	err = writerVertices.Write([]string{"vertex_id", "order_pos", "importance", "geom"})
	if err != nil {
		fmt.Println(err)
		return
	}

	verticesGeoms := make(map[int64]osm2ch.GeoPoint)
	graph := ch.Graph{}

	// Prepare graph and write edges
	for _, edge := range edgeExpandedGraph {
		source := int64(edge.Source)
		target := int64(edge.Target)
		err := graph.CreateVertex(source)
		if err != nil {
			err = errors.Wrap(err, "Can not create source vertex")
			return
		}
		err = graph.CreateVertex(target)
		if err != nil {
			err = errors.Wrap(err, "Can not create source vertex")
			return
		}
		cost := edge.CostMeters
		if strings.ToLower(*units) == "m" {
			cost *= 1000.0
		}
		err = graph.AddEdge(source, target, cost)
		if err != nil {
			err = errors.Wrap(err, "Can not wrap Source and Targed vertices as Edge")
			return
		}
		if len(edge.Geom) < 2 {
			fmt.Println("!!")
			// Skip bad expanded edges
			continue
		}

		geomStr := ""
		if strings.ToLower(*geomFormat) == "geojson" {
			geomStr = osm2ch.PrepareGeoJSONLinestring(edge.Geom)
		} else {
			geomStr = osm2ch.PrepareWKTLinestring(edge.Geom)
		}

		if _, ok := verticesGeoms[source]; !ok {
			verticesGeoms[source] = osm2ch.GeoPoint{Lon: edge.Geom[0].Lon, Lat: edge.Geom[0].Lat}
		}
		if _, ok := verticesGeoms[target]; !ok {
			verticesGeoms[target] = osm2ch.GeoPoint{Lon: edge.Geom[len(edge.Geom)-1].Lon, Lat: edge.Geom[len(edge.Geom)-1].Lat}
		}

		err = writerEdges.Write([]string{
			fmt.Sprintf("%d", source),
			fmt.Sprintf("%d", target),
			fmt.Sprintf("%f", cost),
			geomStr,
			fmt.Sprintf("%t", edge.WasOneway),
			fmt.Sprintf("%d", edge.ID),
			fmt.Sprintf("%d", edge.SourceOSMWayID),
			fmt.Sprintf("%d", edge.TargetOSMWayID),
			fmt.Sprintf("%d", edge.SourceComponent.SourceNodeID), fmt.Sprintf("%d", edge.SourceComponent.TargetNodeID),
			fmt.Sprintf("%d", edge.TargeComponent.SourceNodeID), fmt.Sprintf("%d", edge.TargeComponent.TargetNodeID),
		})

		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if *doContraction {
		fmt.Println("Starting contraction process....")
		st := time.Now()
		graph.PrepareContractionHierarchies()
		fmt.Printf("Done contraction process in %v\n", time.Since(st))
	}

	/* Write vertices */
	vertices := graph.Vertices
	for i := 0; i < len(vertices); i++ {
		currentVertexExternal := vertices[i].Label
		vertexGeom := verticesGeoms[currentVertexExternal]
		geomStr := ""
		if strings.ToLower(*geomFormat) == "geojson" {
			geomStr = osm2ch.PrepareGeoJSONPoint(vertexGeom)
		} else {
			geomStr = osm2ch.PrepareWKTPoint(vertexGeom)
		}
		// Write reference information about vertex
		err = writerVertices.Write([]string{
			fmt.Sprintf("%d", currentVertexExternal),
			fmt.Sprintf("%d", graph.Vertices[i].OrderPos()),
			fmt.Sprintf("%d", graph.Vertices[i].Importance()),
			fmt.Sprintf("%s", geomStr),
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if *doContraction {
		/* Write shortcuts */
		// 	from_vertex_id - int64, ID of source vertex
		// 	to_vertex_id - int64, ID of arget vertex
		// 	weight - float64, Weight of an edge
		// 	via_vertex_id - int64, ID of vertex through which the shortcut exists
		err = graph.ExportShortcutsToFile(fnameShortcuts)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
