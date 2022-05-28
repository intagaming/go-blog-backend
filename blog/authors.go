package blog

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"hxann.com/blog/models"
)

type authorCtxKey struct{}

func (env *Env) AuthorContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var author *models.Author
		var err error

		if userId := chi.URLParam(r, "user_id"); userId != "" {
			author, err = env.authors.Get(userId)
		} else {
			render.Render(w, r, ErrInvalidRequest(errors.New("user_id required")))
		}
		if err == sql.ErrNoRows {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrInternal(err))
			panic(err)
		}
		ctx := context.WithValue(r.Context(), authorCtxKey{}, author)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (env *Env) AuthorGet(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	resp := NewAuthorResponse(author)

	render.Render(w, r, resp)
}

func (env *Env) AuthorPut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	data := &AuthorRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	newAuthor := data.Author
	if newAuthor == nil {
		newAuthor = &models.Author{}
	}
	newAuthor.UserId = author.UserId
	// Fill in missing fields
	if newAuthor.FullName == "" {
		newAuthor.FullName = author.FullName
	}
	if newAuthor.Email == "" {
		newAuthor.Email = author.Email
	}
	if newAuthor.Bio == "" {
		newAuthor.Bio = author.Bio
	}

	err := env.authors.Update(newAuthor)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	resp := NewAuthorResponse(newAuthor)

	render.Render(w, r, resp)
}

func (env *Env) AuthorsMeGet(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(requestAuthorCtxKey{}).(*models.Author)

	resp := NewAuthorResponse(author)

	render.Render(w, r, resp)
}

func (env *Env) AuthorsMePut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(requestAuthorCtxKey{}).(*models.Author)

	data := &AuthorRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	newAuthor := data.Author
	if newAuthor == nil {
		newAuthor = &models.Author{}
	}
	newAuthor.UserId = author.UserId
	// Fill in missing fields
	if newAuthor.FullName == "" {
		newAuthor.FullName = author.FullName
	}
	if newAuthor.Email == "" {
		newAuthor.Email = author.Email
	}
	if newAuthor.Bio == "" {
		newAuthor.Bio = author.Bio
	}

	err := env.authors.Update(newAuthor)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	resp := NewAuthorResponse(newAuthor)

	render.Render(w, r, resp)
}

type AuthorRequest struct {
	*models.Author
}

func (ar *AuthorRequest) Bind(r *http.Request) error {
	if ar == nil {
		return errors.New("missing required Author fields")
	}

	return nil
}

type AuthorResponse struct {
	*models.Author
}

func (resp *AuthorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func NewAuthorResponse(author *models.Author) *AuthorResponse {
	resp := &AuthorResponse{Author: author}

	return resp
}

func NewAuthorListResponse(authors []*models.Author) []*AuthorResponse {
	list := []*AuthorResponse{}
	for _, author := range authors {
		list = append(list, NewAuthorResponse(author))
	}
	return list
}
