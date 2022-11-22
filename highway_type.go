package osm2ch

type HighwayType uint16

const (
	HIGHWAY_MOTORWAY = HighwayType(iota + 1)
	HIGHWAY_MOTORWAY_LINK
	HIGHWAY_TRUNK
	HIGHWAY_TRUNK_LINK
	HIGHWAY_PRIMARY
	HIGHWAY_PRIMARY_LINK
	HIGHWAY_SECONDARY
	HIGHWAY_SECONDARY_LINK
	HIGHWAY_TERTIARY
	HIGHWAY_TERTIARY_LINK
	HIGHWAY_RESIDENTIAL
	HIGHWAY_RESIDENTIAL_LINK
	HIGHWAY_LIVING_STREET
	HIGHWAY_SERVICE
	HIGHWAY_SERVICES
	HIGHWAY_CYCLEWAY
	HIGHWAY_FOOTWAY
	HIGHWAY_PEDESTRIAN
	HIGHWAY_STEPS
	HIGHWAY_TRACK
	HIGHWAY_UNCLASSIFIED
)

func (iotaIdx HighwayType) String() string {
	return [...]string{"motorway", "motorway_link", "trunk", "trunk_link", "primary", "primary_link", "secondary", "secondary_link", "tertiary", "tertiary_link", "residential", "residential_link", "living_street", "service", "services", "cycleway", "footway", "pedestrian", "steps", "track", "unclassified"}[iotaIdx-1]
}

func getHighwayType(str string) HighwayType {
	if found, ok := highwaysTypes[str]; ok {
		return found
	}
	return 0
}

var (
	linkTypeByHighway = map[HighwayType]linkComposition{
		HIGHWAY_MOTORWAY:         {LINK_MOTORWAY, NOT_A_LINK},
		HIGHWAY_MOTORWAY_LINK:    {LINK_MOTORWAY, IS_LINK},
		HIGHWAY_TRUNK:            {LINK_TRUNK, NOT_A_LINK},
		HIGHWAY_TRUNK_LINK:       {LINK_TRUNK, IS_LINK},
		HIGHWAY_PRIMARY:          {LINK_PRIMARY, NOT_A_LINK},
		HIGHWAY_PRIMARY_LINK:     {LINK_PRIMARY, IS_LINK},
		HIGHWAY_SECONDARY:        {LINK_SECONDARY, NOT_A_LINK},
		HIGHWAY_SECONDARY_LINK:   {LINK_SECONDARY, IS_LINK},
		HIGHWAY_TERTIARY:         {LINK_TERTIARY, NOT_A_LINK},
		HIGHWAY_TERTIARY_LINK:    {LINK_TERTIARY, IS_LINK},
		HIGHWAY_RESIDENTIAL:      {LINK_RESIDENTIAL, NOT_A_LINK},
		HIGHWAY_RESIDENTIAL_LINK: {LINK_RESIDENTIAL, IS_LINK},
		HIGHWAY_LIVING_STREET:    {LINK_LIVING_STREET, NOT_A_LINK},
		HIGHWAY_SERVICE:          {LINK_SERVICE, NOT_A_LINK},
		HIGHWAY_SERVICES:         {LINK_SERVICE, NOT_A_LINK},
		HIGHWAY_CYCLEWAY:         {LINK_CYCLEWAY, NOT_A_LINK},
		HIGHWAY_FOOTWAY:          {LINK_FOOTWAY, NOT_A_LINK},
		HIGHWAY_PEDESTRIAN:       {LINK_FOOTWAY, NOT_A_LINK},
		HIGHWAY_STEPS:            {LINK_FOOTWAY, NOT_A_LINK},
		HIGHWAY_TRACK:            {LINK_TRACK, NOT_A_LINK},
		HIGHWAY_UNCLASSIFIED:     {LINK_UNCLASSIFIED, NOT_A_LINK},
	}

	highwaysTypes = map[string]HighwayType{
		"motorway":         HIGHWAY_MOTORWAY,
		"motorway_link":    HIGHWAY_MOTORWAY_LINK,
		"trunk":            HIGHWAY_TRUNK,
		"trunk_link":       HIGHWAY_TRUNK_LINK,
		"primary":          HIGHWAY_PRIMARY,
		"primary_link":     HIGHWAY_PRIMARY_LINK,
		"secondary":        HIGHWAY_SECONDARY,
		"secondary_link":   HIGHWAY_SECONDARY_LINK,
		"tertiary":         HIGHWAY_TERTIARY,
		"tertiary_link":    HIGHWAY_TERTIARY_LINK,
		"residential":      HIGHWAY_RESIDENTIAL,
		"residential_link": HIGHWAY_RESIDENTIAL_LINK,
		"living_street":    HIGHWAY_LIVING_STREET,
		"service":          HIGHWAY_SERVICE,
		"services":         HIGHWAY_SERVICES,
		"cycleway":         HIGHWAY_CYCLEWAY,
		"footway":          HIGHWAY_FOOTWAY,
		"pedestrian":       HIGHWAY_PEDESTRIAN,
		"steps":            HIGHWAY_STEPS,
		"track":            HIGHWAY_TRACK,
		"unclassified":     HIGHWAY_UNCLASSIFIED,
	}
)
