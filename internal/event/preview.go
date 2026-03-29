package event

import "encoding/json"

// PreviewMapper transforms full event payloads into preview/searchable form.
type PreviewMapper interface {
	ToPreview(payload json.RawMessage) (json.RawMessage, error)
	ToSearchableText(payload json.RawMessage) string
}

// PreviewRegistry maps event types to their PreviewMapper.
type PreviewRegistry struct {
	mappers map[string]PreviewMapper
}

func NewPreviewRegistry() *PreviewRegistry {
	return &PreviewRegistry{mappers: make(map[string]PreviewMapper)}
}

func (r *PreviewRegistry) Register(eventType string, mapper PreviewMapper) {
	r.mappers[eventType] = mapper
}

func (r *PreviewRegistry) Get(eventType string) PreviewMapper {
	return r.mappers[eventType]
}

// BuildPreview creates a Preview from an Event using the registered mapper.
func (r *PreviewRegistry) BuildPreview(ev Event) Preview {
	p := Preview{
		UUID:      ev.UUID,
		Type:      ev.Type,
		Payload:   ev.Payload,
		Timestamp: ev.Timestamp,
		Project:   ev.Project,
		IsPinned:  ev.IsPinned,
	}

	mapper := r.Get(ev.Type)
	if mapper != nil {
		if preview, err := mapper.ToPreview(ev.Payload); err == nil {
			p.Payload = preview
		}
		p.SearchableText = mapper.ToSearchableText(ev.Payload)
	}

	return p
}
