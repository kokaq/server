package types

// swagger:model
// swagger:parameters batchEnqueue
// in: body
// required: true
// schema:
type BatchMessageRequest struct {
	Messages []MessageRequest `json:"messages"`
}
