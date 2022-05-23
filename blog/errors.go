package blog

import (
	"net/http"

	"github.com/go-chi/render"
)

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (err *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, err.Status)
	return nil
}

func ErrInternal(err error) render.Renderer {
	return &ErrorResponse{
		Status:  http.StatusInternalServerError,
		Message: err.Error(),
	}
}
