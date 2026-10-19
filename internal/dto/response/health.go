package response

// HealthResponse defines the liveness response.
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse defines the readiness response.
type ReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
