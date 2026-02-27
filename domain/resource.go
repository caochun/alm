package domain

// ResourceSpec defines compute resource requirements for a service or infrastructure component.
type ResourceSpec struct {
	CPU      string // e.g., "1", "0.5", "2" (number of cores)
	Memory   string // e.g., "512Mi", "2Gi"
	Storage  string // e.g., "20Gi" (optional, for stateful resources)
	Replicas int    // number of instances; 0 means unset (defaults to 1)
}

// VolumeSpec defines a persistent volume mount.
type VolumeSpec struct {
	Name  string
	Size  string
	Mount string // container path to mount the volume
}
