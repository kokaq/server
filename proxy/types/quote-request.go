package types

// swagger:model
// swagger:parameters setQuotas
// in: body
// required: true
// schema:
type QuotaRequest struct {
	MaxQueues int `json:"max_queues"`
}
