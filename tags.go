package osm2ch

type AgentType uint16

const (
	AGENT_AUTO = AgentType(iota + 1)
	AGENT_BIKE
	AGENT_WALK
)

func (iotaIdx AgentType) String() string {
	return [...]string{"auto", "bike", "walk"}[iotaIdx-1]
}

type AccessType uint16

const (
	ACCESS_HIGHWAY = AccessType(iota + 1)
	ACCESS_MOTOR_VEHICLE
	ACCESS_MOTORCAR
	ACCESS_OSM_ACCESS
	ACCESS_SERVICE
	ACCESS_BICYCLE
	ACCESS_FOOT
)

func (iotaIdx AccessType) String() string {
	return [...]string{"highway", "motor_vehicle", "motorcar", "access", "service", "bicycle", "foot"}[iotaIdx-1]
}

var (
	networkTypes = map[string]struct{}{
		"auto":    {},
		"bike":    {},
		"walk":    {},
		"railway": {},
		"aeroway": {},
	}

	agentTypes = map[AgentType]struct{}{
		AGENT_AUTO: {},
		AGENT_BIKE: {},
		AGENT_WALK: {},
	}

	agentFiltersInclude = map[AgentType]map[AccessType]map[string]struct{}{
		AGENT_AUTO: {
			ACCESS_MOTOR_VEHICLE: {
				"yes": struct{}{},
			},
			ACCESS_MOTORCAR: {
				"yes": struct{}{},
			},
		},
		AGENT_BIKE: {
			ACCESS_BICYCLE: {
				"yes": struct{}{},
			},
		},
		AGENT_WALK: {
			ACCESS_FOOT: {
				"yes": struct{}{},
			},
		},
	}

	agentFiltersExclude = map[AgentType]map[AccessType]map[string]struct{}{
		AGENT_AUTO: {
			ACCESS_HIGHWAY: {
				"cycleway":      struct{}{},
				"footway":       struct{}{},
				"pedestrian":    struct{}{},
				"steps":         struct{}{},
				"track":         struct{}{},
				"corridor":      struct{}{},
				"elevator":      struct{}{},
				"escalator":     struct{}{},
				"service":       struct{}{},
				"living_street": struct{}{},
			},
			ACCESS_MOTOR_VEHICLE: {
				"no": struct{}{},
			},
			ACCESS_MOTORCAR: {
				"no": struct{}{},
			},
			ACCESS_OSM_ACCESS: {
				"private": struct{}{},
			},
			ACCESS_SERVICE: {
				"parking":          struct{}{},
				"parking_aisle":    struct{}{},
				"driveway":         struct{}{},
				"private":          struct{}{},
				"emergency_access": struct{}{},
			},
		},
		AGENT_BIKE: {
			ACCESS_HIGHWAY: {
				"footway":       struct{}{},
				"steps":         struct{}{},
				"corridor":      struct{}{},
				"elevator":      struct{}{},
				"escalator":     struct{}{},
				"motor":         struct{}{},
				"motorway":      struct{}{},
				"motorway_link": struct{}{},
			},
			ACCESS_BICYCLE: {
				"no": struct{}{},
			},
			ACCESS_SERVICE: {
				"private": struct{}{},
			},
			ACCESS_OSM_ACCESS: {
				"private": struct{}{},
			},
		},
		AGENT_WALK: {
			ACCESS_HIGHWAY: {
				"cycleway":      struct{}{},
				"motor":         struct{}{},
				"motorway":      struct{}{},
				"motorway_link": struct{}{},
			},
			ACCESS_FOOT: {
				"no": struct{}{},
			},
			ACCESS_SERVICE: {
				"private": struct{}{},
			},
			ACCESS_OSM_ACCESS: {
				"private": struct{}{},
			},
		},
	}

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
