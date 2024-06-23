package urls

import "github.com/sajoniks/GoShort/internal/api/v1/event"

const (
	EventTagUrlAdded    = "url_add"
	EventTagUrlAccessed = "url_access"
)

type AddedEvent struct {
	event.BaseEvent
	Source string `json:"source"`
	Alias  string `json:"alias"`
}

type AccessedEvent struct {
	event.BaseEvent
	URL   string `json:"url"`
	Alias string `json:"alias"`
}

func NewAddedEvent(src, alias string) AddedEvent {
	return AddedEvent{
		BaseEvent: event.BaseEvent{
			Type: EventTagUrlAdded,
		},
		Source: src,
		Alias:  alias,
	}
}

func NewAccessedEvent(url, alias string) AccessedEvent {
	return AccessedEvent{
		BaseEvent: event.BaseEvent{
			Type: EventTagUrlAccessed,
		},
		URL:   url,
		Alias: alias,
	}
}
