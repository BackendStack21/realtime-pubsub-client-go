package realtime_pubsub

// PublishOptions represents options for publishing messages.
type PublishOptions struct {
	ID          string
	MessageType string
	Compress    bool
}

// PublishOption defines a function type for setting PublishOptions.
type PublishOption func(*PublishOptions)

// WithPublishID sets the ID in PublishOptions.
func WithPublishID(id string) PublishOption {
	return func(opts *PublishOptions) {
		opts.ID = id
	}
}

// WithPublishMessageType sets the MessageType in PublishOptions.
func WithPublishMessageType(messageType string) PublishOption {
	return func(opts *PublishOptions) {
		opts.MessageType = messageType
	}
}

// WithPublishCompress sets the Compress flag in PublishOptions.
func WithPublishCompress(compress bool) PublishOption {
	return func(opts *PublishOptions) {
		opts.Compress = compress
	}
}

// SendOptions represents options for sending messages.
type SendOptions struct {
	ID          string
	MessageType string
	Compress    bool
}

// SendOption defines a function type for setting SendOptions.
type SendOption func(*SendOptions)

// WithSendID sets the ID in SendOptions.
func WithSendID(id string) SendOption {
	return func(opts *SendOptions) {
		opts.ID = id
	}
}

// WithSendMessageType sets the MessageType in SendOptions.
func WithSendMessageType(messageType string) SendOption {
	return func(opts *SendOptions) {
		opts.MessageType = messageType
	}
}

// WithSendCompress sets the Compress flag in SendOptions.
func WithSendCompress(compress bool) SendOption {
	return func(opts *SendOptions) {
		opts.Compress = compress
	}
}

// ReplyOptions represents options for replying to messages.
type ReplyOptions struct {
	Compress bool
}

// ReplyOption defines a function type for setting ReplyOptions.
type ReplyOption func(*ReplyOptions)

// WithReplyCompress sets the Compress flag in ReplyOptions.
func WithReplyCompress(compress bool) ReplyOption {
	return func(opts *ReplyOptions) {
		opts.Compress = compress
	}
}
