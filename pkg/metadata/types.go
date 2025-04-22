package metadata

// Interface implements GetInstanceID & GetAvailabilityZone
type Interface interface {
	InstanceID() (string, error)
	AZ() (string, error)
}
type Metadata struct {
	UUID             string
	AvailabilityZone string `json:"availabilityZone"`
}
