package orchestrator

import (
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MsgTypeDecision    MessageType = "decision"
	MsgTypeArtifact    MessageType = "artifact"
	MsgTypeReview      MessageType = "review"
	MsgTypeReport      MessageType = "report"
	MsgTypeTask        MessageType = "task"
	MsgTypeActionItem  MessageType = "action_item"
	MsgTypeEscalation  MessageType = "escalation"
	MsgTypeCoordination MessageType = "coordination"
	MsgTypeQuery       MessageType = "query"
	MsgTypeResponse    MessageType = "response"
)

type StructuredMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      MessageType            `json:"type"`
	Subject   string                 `json:"subject"`
	Body      string                 `json:"body"`
	Artifacts []MessageArtifact      `json:"artifacts,omitempty"`
	Decisions []MessageDecision      `json:"decisions,omitempty"`
	Actions   []MessageAction        `json:"actions,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	InReplyTo string                 `json:"in_reply_to,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

type MessageArtifact struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Size int    `json:"size"`
}

type MessageDecision struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type MessageAction struct {
	Description string `json:"description"`
	Assignee    string `json:"assignee"`
	Priority    string `json:"priority"`
	Done        bool   `json:"done"`
}

type CommunicationHub struct {
	exchange   chan StructuredMessage
	routes     map[string]chan StructuredMessage
}

func NewCommunicationHub() *CommunicationHub {
	return &CommunicationHub{
		exchange: make(chan StructuredMessage, 100),
		routes:   make(map[string]chan StructuredMessage),
	}
}

func (ch *CommunicationHub) Send(msg StructuredMessage) {
	msg.ID = uuid.New().String()
	msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	ch.exchange <- msg
}

func (ch *CommunicationHub) Route(msg StructuredMessage) {
	if ch := ch.routes[msg.To]; ch != nil {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (ch *CommunicationHub) Register(agentID string, buffer int) chan StructuredMessage {
	c := make(chan StructuredMessage, buffer)
	ch.routes[agentID] = c
	return c
}

func (ch *CommunicationHub) Unregister(agentID string) {
	delete(ch.routes, agentID)
}

func (ch *CommunicationHub) Start() {
	go func() {
		for msg := range ch.exchange {
			ch.Route(msg)
		}
	}()
}

func NewDecision(from, to, subject, body string) StructuredMessage {
	return StructuredMessage{
		From:    from,
		To:      to,
		Type:    MsgTypeDecision,
		Subject: subject,
		Body:    body,
		Decisions: []MessageDecision{{Subject: subject, Body: body}},
	}
}

func NewReview(from, to, subject, body string, artifacts []MessageArtifact) StructuredMessage {
	return StructuredMessage{
		From:      from,
		To:        to,
		Type:      MsgTypeReview,
		Subject:   subject,
		Body:      body,
		Artifacts: artifacts,
	}
}

func NewReport(from, to, subject, body string) StructuredMessage {
	return StructuredMessage{
		From:    from,
		To:      to,
		Type:    MsgTypeReport,
		Subject: subject,
		Body:    body,
	}
}
