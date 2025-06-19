package types

// swagger:model
// swagger:parameters setPermissions
// in: body
// required: true
// schema:
type PermissionRequest struct {
	Permissions []string `json:"permissions"`
}
