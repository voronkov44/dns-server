package dns

type ServerRequest struct {
	Server string `json:"server"`
}

type ServersResponse struct {
	Servers []string `json:"servers"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
