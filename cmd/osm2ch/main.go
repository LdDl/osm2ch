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
	tagStr        = flag.String("tags", "motorway,primary,primary_link,road,secondary,secondary_link,residential,tertiary,tertiary_link,unclassified,trunk,trunk_link", "Set of needed tags (separated by commas)")
	osmFileName   = flag.String("file", "my_graph.osm.pbf", "Filename of *.osm.pbf file (it has to be compressed)")
	out           = flag.String("out", "my_graph.csv", "Filename of 'Comma-Separated Values' (CSV) formatted file")
	geomFormat    = flag.String("geomf", "wkt", "Format of output geometry. Expected values: wkt / geojson")
	units         = flag.String("units", "km", "Units of output weights. Expected values: km for kilometers / m for meters")
	doContraction = flag.Bool("contraction", false, "Do contraction? Expected values: false - not / true - yes")
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

	file, err := os.Create(*out)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'

	// 		from_vertex_id - int64, ID of source vertex
	// 		to_vertex_id - int64, ID of arget vertex
	// 		f_internal - int64, Internal ID of source vertex
	// 		t_internal - int64, Internal ID of target vertex
	// 		weight - float64, Weight of an edge
	// 		via_vertex_id - int64, ID of vertex through which the contraction exists (-1 if no contraction)
	// 		v_internal - int64, Internal ID of vertex through which the contraction exists (-1 if no contraction)
	//      geom - geometry (WKT or GeoJSON representation)
	//      was_one_way - if edge was one way
	err = writer.Write([]string{"from_vertex_id", "to_vertex_id", "f_internal", "t_internal", "weight", "via_vertex_id", "v_internal", "geom", "was_one_way"})
	if err != nil {
		fmt.Println(err)
		return
	}

	graph := ch.Graph{}
	for source, targets := range edgeExpandedGraph {
		err := graph.CreateVertex(source)
		if err != nil {
			err = errors.Wrap(err, "Can not create source vertex")
			return
		}
		for target, expEdge := range targets {
			err = graph.CreateVertex(target)
			if err != nil {
				err = errors.Wrap(err, "Can not create Target vertex")
				return
			}
			cost := expEdge.Cost
			if strings.ToLower(*units) == "m" {
				cost *= 1000.0
			}
			err = graph.AddEdge(source, target, cost)
			if err != nil {
				err = errors.Wrap(err, "Can not wrap Source and Targed vertices as Edge")
				return
			}
		}
	}

	if *doContraction {
		fmt.Println("Starting contraction process....")
		st := time.Now()
		graph.PrepareContracts()
		fmt.Printf("Done contraction process in %v\n", time.Since(st))
	}

	vertices := graph.Vertices
	for i := 0; i < len(vertices); i++ {
		currentVertexExternal := vertices[i].Label
		// currentVertexInternal := graph.Vertices[i].VertexNum()

		_ = currentVertexExternal
	}

	// for source, targets := range edgeExpandedGraph {
	// 	for target, expEdge := range targets {

	// 		// 		from_vertex_id - int64, ID of source vertex
	// 		// 		to_vertex_id - int64, ID of arget vertex
	// 		// 		f_internal - int64, Internal ID of source vertex
	// 		// 		t_internal - int64, Internal ID of target vertex
	// 		// 		weight - float64, Weight of an edge
	// 		// 		via_vertex_id - int64, ID of vertex through which the contraction exists (-1 if no contraction)
	// 		// 		v_internal - int64, Internal ID of vertex through which the contraction exists (-1 if no contraction)
	// 		//      geom - geometry (WKT or GeoJSON representation)
	// 		//      was_one_way - if edge was one way

	// 		geomStr := ""
	// 		if strings.ToLower(*geomFormat) == "geojson" {
	// 			geomStr = osm2ch.PrepareGeoJSONLinestring(expEdge.Geom)
	// 		} else {
	// 			geomStr = osm2ch.PrepareWKTLinestring(expEdge.Geom)
	// 		}
	// 		cost := expEdge.Cost
	// 		if strings.ToLower(*units) == "m" {
	// 			cost *= 1000.0
	// 		}
	// 		err = writer.Write([]string{fmt.Sprintf("%d", source), fmt.Sprintf("%d", target), "FT", fmt.Sprintf("%f", cost), geomStr, strconv.FormatBool(expEdge.WasOneWay)})
	// 		if err != nil {
	// 			fmt.Println(err)
	// 			return
	// 		}
	// 	}
	// }
}
