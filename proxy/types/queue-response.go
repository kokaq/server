package types

// swagger:model
// QueueResponse defines output for queue operations
type QueueResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxSize     int    `json:"max_size"`
}
