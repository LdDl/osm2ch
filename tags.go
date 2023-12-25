package osm2ch

var (
	junctionTypes = map[string]struct{}{
		"circular":   {},
		"roundabout": {},
	}

	poiHighwayTags = map[string]struct{}{
		"bus_stop": {},
		"platform": {},
	}

	poiRailwayTags = map[string]struct{}{
		"depot":         {},
		"workshop":      {},
		"halt":          {},
		"interlocking":  {},
		"junction":      {},
		"spur_junction": {},
		"terminal":      {},
		"platform":      {},
	}

	poiAerowayTags = map[string]struct{}{}

	negligibleHighwayTags = map[string]struct{}{
		"path":         {},
		"construction": {},
		"proposed":     {},
		"raceway":      {},
		"bridleway":    {},
		"rest_area":    {},
		"su":           {},
		"road":         {},
		"abandoned":    {},
		"planned":      {},
		"trailhead":    {},
		"stairs":       {},
		"dismantled":   {},
		"disused":      {},
		"razed":        {},
		"access":       {},
		"corridor":     {},
		"stop":         {},
	}

	// See ref.: https://wiki.openstreetmap.org/wiki/Tag:oneway%3Dreversible
	onewayReversible = map[string]struct{}{
		"reversible":  {},
		"alternating": {},
	}
)
