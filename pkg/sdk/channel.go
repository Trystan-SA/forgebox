package sdk

import "context"

// ChannelPlugin is the interface for messaging platform integrations.
//
// Channels connect ForgeBox to communication platforms like Slack, Discord,
// Microsoft Teams, and Telegram. They receive inbound messages and send
// results back to users.
type ChannelPlugin interface {
	Plugin

	// Listen starts listening for inbound messages on this channel.
	// It blocks until ctx is canceled. Received messages are dispatched
	// to the handler.
	Listen(ctx context.Context, handler MessageHandler) error

	// Send delivers an outbound message to the channel.
	Send(ctx context.Context, msg *OutboundMessage) error
}

// MessageHandler processes inbound messages from a channel.
type MessageHandler func(ctx context.Context, msg *InboundMessage) error

// InboundMessage is a message received from a messaging platform.
type InboundMessage struct {
	// ChannelName identifies which channel plugin received this message.
	ChannelName string `json:"channel_name"`

	// ChannelID is the platform-specific channel/room identifier.
	ChannelID string `json:"channel_id"`

	// UserID is the platform-specific user identifier.
	UserID string `json:"user_id"`

	// UserName is the display name of the user.
	UserName string `json:"user_name"`

	// Text is the message content.
	Text string `json:"text"`

	// ThreadID groups messages into a conversation thread.
	ThreadID string `json:"thread_id,omitempty"`

	// Metadata contains platform-specific extra data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// OutboundMessage is a message to send to a messaging platform.
type OutboundMessage struct {
	ChannelName string `json:"channel_name"`
	ChannelID   string `json:"channel_id"`
	ThreadID    string `json:"thread_id,omitempty"`
	Text        string `json:"text"`
	// Blocks contains rich formatting (Slack blocks, Discord embeds, etc.).
	Blocks []any `json:"blocks,omitempty"`
}
