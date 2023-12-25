package osm2ch

type LinkClass uint16

const (
	LINK_CLASS_HIGHWAY = LinkClass(iota + 1)
	LINK_CLASS_RAILWAY
	LINK_CLASS_AEROWAY
)

func (iotaIdx LinkClass) String() string {
	return [...]string{"highway", "railway", "aeroway"}[iotaIdx-1]
}
