package osm2ch

var (
	// default_lanes_dict = map[string]int{"motorway": 4, "trunk": 3, "primary": 3, "secondary": 2, "tertiary": 2,
	// 	"residential": 1, "service": 1, "cycleway": 1, "footway": 1, "track": 1,
	// 	"unclassified": 1, "connector": 2}
	// default_speed_dict = map[string]float64{"motorway": 120, "trunk": 100, "primary": 80, "secondary": 60, "tertiary": 40,
	// 	"residential": 30, "service": 30, "cycleway": 5, "footway": 5, "track": 30,
	// 	"unclassified": 30, "connector": 120}
	// default_capacity_dict = map[string]int{"motorway": 2300, "trunk": 2200, "primary": 1800, "secondary": 1600, "tertiary": 1200,
	// 	"residential": 1000, "service": 800, "cycleway": 800, "footway": 800, "track": 800,
	// 	"unclassified": 800, "connector": 9999}

	// poiTags = map[string]struct{}{
	// 	"building": struct{}{},
	// 	"amenity":  struct{}{},
	// 	"leisure":  struct{}{},
	// }

	junctionTypes = map[string]struct{}{
		"circular":   struct{}{},
		"roundabout": struct{}{},
	}

	poiHighwayTags = map[string]struct{}{
		"bus_stop": struct{}{},
		"platform": struct{}{},
	}

	poiRailwayTags = map[string]struct{}{
		"depot":         struct{}{},
		"workshop":      struct{}{},
		"halt":          struct{}{},
		"interlocking":  struct{}{},
		"junction":      struct{}{},
		"spur_junction": struct{}{},
		"terminal":      struct{}{},
		"platform":      struct{}{},
	}

	poiAerowayTags = map[string]struct{}{}

	negligibleHighwayTags = map[string]struct{}{
		"path":         struct{}{},
		"construction": struct{}{},
		"proposed":     struct{}{},
		"raceway":      struct{}{},
		"bridleway":    struct{}{},
		"rest_area":    struct{}{},
		"su":           struct{}{},
		"road":         struct{}{},
		"abandoned":    struct{}{},
		"planned":      struct{}{},
		"trailhead":    struct{}{},
		"stairs":       struct{}{},
		"dismantled":   struct{}{},
		"disused":      struct{}{},
		"razed":        struct{}{},
		"access":       struct{}{},
		"corridor":     struct{}{},
		"stop":         struct{}{},
	}

	// See ref.: https://wiki.openstreetmap.org/wiki/Tag:oneway%3Dreversible
	onewayReversible = map[string]struct{}{
		"reversible":  struct{}{},
		"alternating": struct{}{},
	}
)
