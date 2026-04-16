[![GoDoc](https://godoc.org/github.com/LdDl/osm2ch?status.svg)](https://godoc.org/github.com/LdDl/osm2ch)
[![Build Status](https://travis-ci.com/LdDl/osm2ch.svg?branch=master)](https://travis-ci.com/LdDl/osm2ch)
[![Sourcegraph](https://sourcegraph.com/github.com/LdDl/osm2ch/-/badge.svg)](https://sourcegraph.com/github.com/LdDl/osm2ch?badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/LdDl/osm2ch)](https://goreportcard.com/report/github.com/LdDl/osm2ch)
[![GitHub tag](https://img.shields.io/github/tag/LdDl/osm2ch.svg)](https://github.com/LdDl/osm2ch/releases)
# osm2ch
## Convert *.osm.pbf files to CSV for [contraction hierarchies library](https://github.com/LdDl/ch)

- [osm2ch](#osm2ch)
  - [About](#about)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Example](#example)
  - [Output format](#output-format)
  - [Dependencies](#dependencies)
  - [License](#license)

## About
With this CLI tool you can convert *.osm.pbf (Compressed Open Street Map) file to CSV (Comma-Separated Values) file, which is used in our [contraction hierarchies library].

Under the hood it uses [`osm2gmns`](https://github.com/LdDl/osm2gmns) library (Go port of [osm2gmns](https://osm2gmns.readthedocs.io/en/stable/)) to build a macroscopic graph with turning movements, then this CLI performs edge expansion and contraction hierarchies.

What it does:
- Builds macroscopic network from OSM via [`osm2gmns`](https://github.com/LdDl/osm2gmns);
- Generates turning movements at intersections;
- Performs edge expansion (each macro link becomes a vertex, movements become edges);
- Filters U-turns;
- Supported types of restrictions (internally processed in `osm2gmns`):
      - only_left_turn;
      - only_right_turn;
      - only_straight_on;
      - no_left_turn;
      - no_right_turn;
      - no_straight_on.
- Prepares contraction hierarchies;
- Outputs CSV files with geometry in WKT or GeoJSON format.

Agent type filtering is handled by `osm2gmns` (e.g. `auto`, `bike`, `walk`).

PRs are welcome!

## Installation
* Via 'go install':
    ```shell
    go install github.com/LdDl/osm2ch/cmd/osm2ch@latest
    ```
    After installation step is complete you can call 'osm2ch' from any place in your system.

* Or download prebuilt binary from [Releases](https://github.com/LdDl/osm2ch/releases) and add it to your PATH.

* Docker:
    ```shell
    docker pull dimahkiin/osm2ch:latest
    ```

## Usage

```shell
osm2ch -h
```

Output:
```shell
Usage of osm2ch:
  -file string
        Filename of *.osm.pbf file (it has to be compressed) (default "my_graph.osm.pbf")
  -out string
        Filename of 'Comma-Separated Values' (CSV) formatted file (default "my_graph.csv")
        E.g.: if file name is 'map.csv' then 3 files will be produced: 'map.csv' (edges), 'map_vertices.csv', 'map_shortcuts.csv'
  -geomf string
        Format of output geometry. Expected values: wkt / geojson (default "wkt")
  -agents string
        Comma-separated list of agent types. Expected values: auto,bike,walk (default "auto")
```

## Example
You can find example file of *.osm.pbf file in nested child [/example_data](/example_data).

If you want WKT format for output geometry and `auto` agent type:
```shell
osm2ch --file example_data/moscow_center_reduced.osm.pbf --out graph.csv --geomf wkt --agents auto
```

If you want GeoJSON format for output geometry and `auto` agent type:
```shell
osm2ch --file example_data/moscow_center_reduced.osm.pbf --out graph.csv --geomf geojson --agents auto
```

After that 3 files will be created: `graph.csv` (edges), `graph_vertices.csv`, `graph_shortcuts.csv`.

## Output format

### Edges CSV

Header: `from_vertex_id;to_vertex_id;length_meters;free_speed;geom;was_one_way;edge_id;osm_way_from;osm_way_to;osm_way_from_source_node;osm_way_from_target_node;osm_way_to_source_node;osm_way_to_target_node;macro_node_from_source;macro_node_from_target;macro_node_to_source;macro_node_to_target`

| Column | Type | Description |
|--------|------|-------------|
| from_vertex_id | int64 | Source vertex in edge-expanded graph (= source macro link ID) |
| to_vertex_id | int64 | Target vertex in edge-expanded graph (= target macro link ID) |
| length_meters | float64 | Length of the expanded edge in meters (sum of the second half of the incoming macro link and the first half of the outgoing one) |
| free_speed | float64 | Free-flow speed in km/h - see note below |
| geom | string | Geometry (LineString) in WKT or GeoJSON format |
| was_one_way | bool | True if the source OSM way was one-way |
| edge_id | int64 | ID of the expanded edge |
| osm_way_from | int64 | OSM Way ID of the source (income) macro link |
| osm_way_to | int64 | OSM Way ID of the target (outcome) macro link |
| osm_way_from_source_node | int64 | OSM Node ID at the start of the source macro link |
| osm_way_from_target_node | int64 | OSM Node ID at the end of the source macro link |
| osm_way_to_source_node | int64 | OSM Node ID at the start of the target macro link |
| osm_way_to_target_node | int64 | OSM Node ID at the end of the target macro link |
| macro_node_from_source | int | GMNS macro node ID at the start of the source macro link |
| macro_node_from_target | int | GMNS macro node ID at the end of the source macro link |
| macro_node_to_source | int | GMNS macro node ID at the start of the target macro link |
| macro_node_to_target | int | GMNS macro node ID at the end of the target macro link |

### Vertices CSV

Header: `vertex_id;order_pos;importance;geom`

| Column | Type | Description |
|--------|------|-------------|
| vertex_id | int64 | Vertex ID (= macro link ID) |
| order_pos | int | Order position in contraction hierarchies |
| importance | int | Importance of vertex in contraction hierarchies |
| geom | string | Geometry (Point) in WKT or GeoJSON format |

### Shortcuts CSV

Header: `from_vertex_id;to_vertex_id;weight;via_vertex_id`

| Column | Type | Description |
|--------|------|-------------|
| from_vertex_id | int64 | Source vertex |
| to_vertex_id | int64 | Target vertex |
| weight | float64 | Shortcut cost in **seconds** (travel time) |
| via_vertex_id | int64 | Intermediate vertex |

Contraction hierarchies are built using **travel time** (seconds) as the cost metric: `length_meters / (free_speed_kmh / 3.6)`. Shortcut costs are also in seconds.

> **Note on `free_speed`.**
> An expanded edge is assembled from two halves of two adjacent macro links (the second half of the incoming link + the first half of the outgoing one), which may have different free-flow speeds. To keep the total travel time physically consistent, `free_speed` is computed as a **length-weighted harmonic mean** of the two macro speeds:
>
> ```
> free_speed = (inHalf + outHalf) / (inHalf / inSpeed + outHalf / outSpeed)
> ```
>
> This guarantees that `length_meters / free_speed` equals the sum of the travel times across the two halves. A plain arithmetic mean would over-estimate the effective speed when the two halves differ significantly (e.g. a `motorway_link` at 30 km/h followed by a `motorway` at 100 km/h). If one of the macro links has a missing / zero speed, the other one is used as-is.
>
> **Worked example.** Suppose the incoming macro link is a `motorway_link` of length 60 m at 30 km/h, and the outgoing one is a `motorway` of length 100 m at 100 km/h. Then:
>
> ```
> inHalf  = 60 / 2  = 30 m
> outHalf = 100 / 2 = 50 m
> totalLength = 30 + 50 = 80 m
> ```
>
> True travel time across the expanded edge (physically correct reference):
>
> ```
> t_in  = 30 m / (30  km/h / 3.6) = 30 * 3.6 /  30 = 3.6 s
> t_out = 50 m / (100 km/h / 3.6) = 50 * 3.6 / 100 = 1.8 s
> t_true = t_in + t_out = 5.4 s
> ```
>
> Arithmetic mean (the old formula) would give `free_speed = (30 + 100) / 2 = 65 km/h`, implying a travel time of `80 * 3.6 / 65 ~= 4.43 s` - which is under-estimated by approximately 18%.
>
> Length-weighted harmonic mean (the current formula):
>
> ```
> free_speed = (30 + 50) / (30/30 + 50/100)
>            = 80 / (1.0 + 0.5)
>            = 80 / 1.5
>            ~= 53.33 km/h
> ```
>
> Sanity-check: `80 * 3.6 / 53.33 = 5.4 s` - matches `t_true` exactly, as required.

If you need route distance in meters (e.g. for map matching transition probabilities), sum up `length_meters` of each edge along the CH path.

Now you can use this graph in [contraction hierarchies library].

## Dependencies
- [osm2gmns](https://github.com/LdDl/osm2gmns) - OSM to GMNS macroscopic network + movements
- [go-gmns](https://github.com/LdDl/go-gmns) - GMNS data structures and generators
- [ch](https://github.com/LdDl/ch) - Contraction hierarchies
- [paulmach/osm](https://github.com/paulmach/osm) - OSM data types
- [paulmach/orb](https://github.com/paulmach/orb) - Geometry types

## License
- Please see [here](LICENSE.md)

[contraction hierarchies library]: https://github.com/LdDl/ch#ch---contraction-hierarchies
