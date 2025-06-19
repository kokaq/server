package types

// QueueRequest defines input for queue creation/update
// swagger:parameters createQueue updateQueue
// in: body
// required: true
// schema:
type QueueRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxSize     int    `json:"max_size"`
}
