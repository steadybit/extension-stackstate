package extservice

type ViewSnapshotResponseWrapper struct {
	ViewSnapshotResponse ViewSnapshotResponse `json:"viewSnapshotResponse"`
}
type ViewSnapshotResponse struct {
	Components []Component `json:"components"`
}
type Component struct {
	Id          int        `json:"id"`
	Name        string     `json:"name"`
	State       State      `json:"state"`
	Properties  Properties `json:"properties"`
	Identifiers []string   `json:"identifiers"`
}
type State struct {
	HealthState string `json:"healthState"`
}
type Properties struct {
	NamespaceIdentifier   string `json:"namespaceIdentifier"`
	ClusterNameIdentifier string `json:"clusterNameIdentifier"`
}
