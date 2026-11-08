package module

import (
	"reflect"
	"testing"
)

type deleteListenerModule struct {
	BaseModule
	got []string
}

func (m *deleteListenerModule) Name() string                   { return "listener" }
func (m *deleteListenerModule) Type() string                   { return "listener" }
func (m *deleteListenerModule) OnEventsDeleted(uuids []string) { m.got = append(m.got, uuids...) }

func TestRegistry_NotifyEventsDeleted(t *testing.T) {
	r := NewRegistry()
	l := &deleteListenerModule{}
	r.Register(plainModule{})
	r.Register(l)

	r.NotifyEventsDeleted([]string{"a", "b"})

	if want := []string{"a", "b"}; !reflect.DeepEqual(l.got, want) {
		t.Errorf("listener got %v, want %v", l.got, want)
	}
}
