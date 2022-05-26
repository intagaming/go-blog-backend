package blog

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"hxann.com/blog/models"
)

func (env *Env) AuthorsMeGet(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	resp := NewAuthorResponse(author)

	render.Render(w, r, resp)
}

func (env *Env) AuthorsMePut(w http.ResponseWriter, r *http.Request) {
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
	// TODO: improve this
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
