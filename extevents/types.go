package extevents

type StackStateIntakeRequest struct {
	CollectionTimestamp int32                      `json:"collection_timestamp"`
	InternalHostname    string                     `json:"internal_hostname"`
	Events              map[string]StackStateEvent `json:"events"`
}
type StackStateEvent struct {
	Context   Context `json:"context"`
	EventType string  `json:"event_type"`
	MsgTitle  string  `json:"msg_title"`
	MsgText   string  `json:"msg_text"`
	Timestamp int32   `json:"timestamp"`
}

type Context struct {
	Category           string            `json:"category"`
	Data               map[string]string `json:"data"`
	ElementIdentifiers []string          `json:"element_identifiers"`
	Source             string            `json:"source"`
	SourceLinks        []SourceLink      `json:"source_links"`
}

type SourceLink struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

type StackStateIntakeResponse struct {
}
