package blog

import (
	"net/http"

	"github.com/go-chi/render"
)

type ErrorResponse struct {
	Err     error  `json:"-"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (err *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, err.Status)
	return nil
}

func ErrInternal(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusInternalServerError,
		Message: "Internal server error.",
	}
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusBadRequest,
		Message: err.Error(),
	}
}
