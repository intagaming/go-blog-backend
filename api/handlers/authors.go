package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"hxann.com/blog/api/middleware"
	"hxann.com/blog/api/resp"
	"hxann.com/blog/models"
)

type Authors struct {
	authors *models.AuthorModel
}

func (_ *Authors) AuthorGet(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.AuthorCtxKey{}).(*models.Author)

	resp := NewAuthorResponse(author)

	render.Render(w, r, resp)
}

func (a *Authors) AuthorPut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.AuthorCtxKey{}).(*models.Author)

	data := &AuthorRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, resp.ErrInvalidRequest(err))
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

	err := a.authors.Update(newAuthor)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	resp := NewAuthorResponse(newAuthor)

	render.Render(w, r, resp)
}

func (a *Authors) AuthorsMeGet(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.RequestAuthorCtxKey{}).(*models.Author)

	resp := NewAuthorResponse(author)

	render.Render(w, r, resp)
}

func (a *Authors) AuthorsMePut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.RequestAuthorCtxKey{}).(*models.Author)

	data := &AuthorRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, resp.ErrInvalidRequest(err))
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

	err := a.authors.Update(newAuthor)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
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

func NewAuthors(authors *models.AuthorModel) *Authors {
	return &Authors{
		authors: authors,
	}
}
