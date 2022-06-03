// Package resp describes HTTP responses in the form of Chi's render.Renderer.
package resp

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

func ErrBadRequest(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusBadRequest,
		Message: err.Error(),
	}
}

func ErrDuplicate(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusConflict,
		Message: "Entity existed.",
	}
}

var ErrNotFound = &ErrorResponse{
	Status:  http.StatusNotFound,
	Message: "Resource not found.",
}

func ErrNotFoundCustom(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusNotFound,
		Message: err.Error(),
	}
}

func ErrForbidden(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusForbidden,
		Message: err.Error(),
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusUnauthorized,
		Message: err.Error(),
	}
}

func ErrTooManyRequest(err error) render.Renderer {
	return &ErrorResponse{
		Err:     err,
		Status:  http.StatusTooManyRequests,
		Message: err.Error(),
	}
}
