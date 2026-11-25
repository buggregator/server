package explain

import (
	"sync"
	"time"
)

// maxMessages a thread keeps; older ones drop off.
const maxMessages = 200

// Message is one line of a thread: the person's question or an agent's answer.
type Message struct {
	From    string    `json:"from"` // "you" or "agent"
	Role    string    `json:"role,omitempty"`
	Project string    `json:"project,omitempty"`
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
}

// Thread is the conversation with the agents about one event.
type Thread struct {
	EventUUID    string    `json:"event_uuid"`
	Project      string    `json:"project"`
	Conversation string    `json:"conversation"`
	Messages     []Message `json:"messages"`
}

// threads keeps the conversations in memory: a concept, lost on restart.
type threads struct {
	mu      sync.Mutex
	byEvent map[string]*Thread
	byConv  map[string]*Thread
}

func newThreads() *threads {
	return &threads{byEvent: map[string]*Thread{}, byConv: map[string]*Thread{}}
}

// get returns a copy of the event's thread.
func (t *threads) get(eventUUID string) (Thread, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	th, ok := t.byEvent[eventUUID]
	if !ok {
		return Thread{}, false
	}
	return copyThread(th), true
}

// start records a new conversation for the event with the person's first message.
func (t *threads) start(eventUUID, project, conversation string, first Message) Thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	if old, ok := t.byEvent[eventUUID]; ok {
		delete(t.byConv, old.Conversation)
	}
	th := &Thread{EventUUID: eventUUID, Project: project, Conversation: conversation, Messages: []Message{first}}
	t.byEvent[eventUUID] = th
	t.byConv[conversation] = th
	return copyThread(th)
}

// add appends a message to the conversation; false when it is unknown.
func (t *threads) add(conversation string, m Message) (Thread, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	th, ok := t.byConv[conversation]
	if !ok {
		return Thread{}, false
	}
	th.Messages = append(th.Messages, m)
	if len(th.Messages) > maxMessages {
		th.Messages = th.Messages[len(th.Messages)-maxMessages:]
	}
	return copyThread(th), true
}

func (t *threads) remove(eventUUIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, u := range eventUUIDs {
		if th, ok := t.byEvent[u]; ok {
			delete(t.byConv, th.Conversation)
			delete(t.byEvent, u)
		}
	}
}

func copyThread(th *Thread) Thread {
	c := *th
	c.Messages = append([]Message(nil), th.Messages...)
	return c
}
