package blog

import (
	"net/http"

	"hxann.com/blog/models"
)

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
