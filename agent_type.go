package osm2ch

type AgentType uint16

const (
	AGENT_AUTO = AgentType(iota + 1)
	AGENT_BIKE
	AGENT_WALK
	AGENT_UNDEFINED = AgentType(0)
)

func (iotaIdx AgentType) String() string {
	return [...]string{"undefined", "auto", "bike", "walk"}[iotaIdx]
}

var (
	agentTypesAll = map[AgentType]struct{}{
		AGENT_AUTO: {},
		AGENT_BIKE: {},
		AGENT_WALK: {},
	}

	agentsAccessIncludeValues = map[AgentType]map[AccessType]map[string]struct{}{
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

	agentsAccessExcludeValues = map[AgentType]map[AccessType]map[string]struct{}{
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
)
