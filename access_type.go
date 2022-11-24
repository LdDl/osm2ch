package osm2ch

type AccessType uint16

const (
	ACCESS_HIGHWAY = AccessType(iota + 1)
	ACCESS_MOTOR_VEHICLE
	ACCESS_MOTORCAR
	ACCESS_OSM_ACCESS
	ACCESS_SERVICE
	ACCESS_BICYCLE
	ACCESS_FOOT
	ACCESS_UNDEFINED = AccessType(0)
)

func (iotaIdx AccessType) String() string {
	return [...]string{"undefined", "highway", "motor_vehicle", "motorcar", "access", "service", "bicycle", "foot"}[iotaIdx]
}
