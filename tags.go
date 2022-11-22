package osm2ch

var (
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
