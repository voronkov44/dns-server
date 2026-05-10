package req

import (
	"dns-manager/pkg/res"
	"net/http"
)

func HandleBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	body, err := Decode[T](r.Body)
	if err != nil {
		res.Json(w, map[string]string{"error": "invalid request body"}, http.StatusBadRequest)
		return nil, err
	}

	return &body, nil
}
