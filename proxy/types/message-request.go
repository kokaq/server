package types

// swagger:model
// swagger:parameters enqueueMessage
// in: body
// required: true
// schema:
type MessageRequest struct {
	Body string `json:"body"`
	TTL  int    `json:"ttl"`
}
