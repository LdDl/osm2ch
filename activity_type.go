package osm2ch

type ActivityType uint16

const (
	ACTIVITY_POI  = ActivityType(iota + 1)
	ACTIVITY_NONE = ActivityType(0)
)

func (iotaIdx ActivityType) String() string {
	return [...]string{"none", "poi"}[iotaIdx]
}
