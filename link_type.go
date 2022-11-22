package osm2ch

type LinkConnectionType uint16

const (
	// Plain way
	NOT_A_LINK = LinkConnectionType(iota)
	// Connection between two roads
	IS_LINK
)

type LinkClass uint16

const (
	LINK_CLASS_HIGHWAY = LinkClass(iota + 1)
	LINK_CLASS_RAILWAY
	LINK_CLASS_AEROWAY
)

func (iotaIdx LinkClass) String() string {
	return [...]string{"highway", "railway", "aeroway"}[iotaIdx-1]
}

type LinkType uint16

const (
	LINK_MOTORWAY = LinkType(iota + 1)
	LINK_TRUNK
	LINK_PRIMARY
	LINK_SECONDARY
	LINK_TERTIARY
	LINK_RESIDENTIAL
	LINK_LIVING_STREET
	LINK_SERVICE
	LINK_CYCLEWAY
	LINK_FOOTWAY
	LINK_TRACK
	LINK_UNCLASSIFIED
	LINK_CONNECTOR
	LINK_RAILWAY
	LINK_AEROWAY
)

func (iotaIdx LinkType) String() string {
	return [...]string{"motorway", "trunk", "primary", "secondary", "tertiary", "residential", "living_street", "service", "cycleway", "footway", "track", "unclassified", "connector", "railway", "aeroway"}[iotaIdx-1]
}

type linkComposition struct {
	linkType           LinkType
	linkConnectionType LinkConnectionType
}

var (
	onewayDefaultByLink = map[LinkType]bool{
		LINK_MOTORWAY:      false,
		LINK_TRUNK:         false,
		LINK_PRIMARY:       false,
		LINK_SECONDARY:     false,
		LINK_TERTIARY:      false,
		LINK_RESIDENTIAL:   false,
		LINK_LIVING_STREET: false,
		LINK_SERVICE:       false,
		LINK_CYCLEWAY:      true,
		LINK_FOOTWAY:       true,
		LINK_TRACK:         true,
		LINK_UNCLASSIFIED:  false,
		LINK_CONNECTOR:     false,
		LINK_RAILWAY:       true,
		LINK_AEROWAY:       true,
	}
)
