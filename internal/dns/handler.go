package dns

import (
	"dns-manager/pkg/req"
	"dns-manager/pkg/res"
	"errors"
	"net/http"
)

type HandlerDeps struct {
	Service *Service
}

type Handler struct {
	service *Service
}

func NewHandler(router *http.ServeMux, deps HandlerDeps) {
	h := &Handler{
		service: deps.Service,
	}

	router.HandleFunc("GET /healthz", h.HealthCheck())
	router.HandleFunc("GET /dns", h.ListServers())
	router.HandleFunc("POST /dns", h.AddServer())
	router.HandleFunc("DELETE /dns", h.DeleteServer())
}

func (h *Handler) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res.Json(w, HealthResponse{Status: "ok"}, http.StatusOK)
	}
}

func (h *Handler) ListServers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servers, err := h.service.ListServers(r.Context())
		if err != nil {
			res.Json(w, ErrorResponse{Error: "failed to list DNS servers"}, http.StatusInternalServerError)
			return
		}

		res.Json(w, toServersResponse(servers), http.StatusOK)
	}
}

func (h *Handler) AddServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[ServerRequest](w, r)
		if err != nil {
			return
		}

		servers, err := h.service.AddServer(r.Context(), body.Server)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		res.Json(w, toServersResponse(servers), http.StatusCreated)
	}
}

func (h *Handler) DeleteServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readDeleteServerRequest(r)
		if err != nil {
			res.Json(w, ErrorResponse{Error: "invalid request body"}, http.StatusBadRequest)
			return
		}

		servers, err := h.service.DeleteServer(r.Context(), body.Server)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		res.Json(w, toServersResponse(servers), http.StatusOK)
	}
}

func readDeleteServerRequest(r *http.Request) (ServerRequest, error) {
	body, err := req.Decode[ServerRequest](r.Body)
	if err == nil {
		return body, nil
	}

	if !errors.Is(err, req.ErrEmptyBody) {
		return ServerRequest{}, err
	}

	server := r.URL.Query().Get("server")
	if server == "" {
		return ServerRequest{}, req.ErrEmptyBody
	}

	return ServerRequest{
		Server: server,
	}, nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidServerAddress):
		res.Json(w, ErrorResponse{Error: ErrInvalidServerAddress.Error()}, http.StatusBadRequest)
	case errors.Is(err, ErrServerAlreadyExists):
		res.Json(w, ErrorResponse{Error: ErrServerAlreadyExists.Error()}, http.StatusConflict)
	case errors.Is(err, ErrServerNotFound):
		res.Json(w, ErrorResponse{Error: ErrServerNotFound.Error()}, http.StatusNotFound)
	default:
		res.Json(w, ErrorResponse{Error: "internal server error"}, http.StatusInternalServerError)
	}
}

// toServersResponse - преобразует внутреннюю модель в красивый json
func toServersResponse(servers []Server) ServersResponse {
	out := make([]string, 0, len(servers))

	for _, server := range servers {
		out = append(out, server.Address)
	}

	return ServersResponse{
		Servers: out,
	}
}
