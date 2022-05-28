package blog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-sql-driver/mysql"
	"hxann.com/blog/blog/constants"
	"hxann.com/blog/models"
)

func (env *Env) PagesGet(w http.ResponseWriter, r *http.Request) {
	// Fetch pages from db
	modelPages, err := env.pages.All()

	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	pagesResp, err := NewPageListResponse(modelPages, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.RenderList(w, r, pagesResp)
}

func (env *Env) PagesPost(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	data := &PageRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	page := data.Page

	// Check required fields
	if page.Slug == "" || page.Title == "" || page.Excerpt == "" ||
		page.Content == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("some of the required fields are not present. Required fields: slug, title, excerpt, content")))
		return
	}

	authorIds := append(data.Authors, author.UserId)
	authors, missingAuthorId, err := AuthorIdsToAuthors(authorIds, env)
	if err != nil {
		if missingAuthorId != nil {
			render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
			return
		}
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}
	for _, dbAuthor := range authors {
		if dbAuthor.UserId == author.UserId {
			page.Author = dbAuthor
		} else {
			page.Authors = append(page.Authors, dbAuthor)
		}
	}

	if err := env.pages.Add(page); err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == 1062 {
				render.Render(w, r, ErrDuplicate(err))
				return
			}
		}
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	insertedPage, err := env.pages.Get(page.Slug)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
		panic(err)
	}

	resp, err := NewPageResponse(insertedPage, env)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
		panic(err)
	}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, resp)
}

type pageCtxKey struct{}

func (env *Env) PageContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var page *models.Page
		var err error

		if slug := chi.URLParam(r, "slug"); slug != "" {
			page, err = env.pages.Get(slug)
		} else { // slug empty
			render.Render(w, r, ErrInvalidRequest(errors.New("slug required")))
		}
		if err == sql.ErrNoRows {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrInternal(err))
			panic(err)
		}
		ctx := context.WithValue(r.Context(), pageCtxKey{}, page)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (env *Env) PageGet(w http.ResponseWriter, r *http.Request) {
	page := r.Context().Value(pageCtxKey{}).(*models.Page)

	resp, err := NewPageResponse(page, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, resp)
}

func (env *Env) PagePut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	page := r.Context().Value(pageCtxKey{}).(*models.Page)

	data := &PageRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	newPage := data.Page
	if newPage == nil {
		newPage = &models.Page{}
	}
	// Provides the slug from context
	newPage.Slug = page.Slug
	// Fill in missing fields
	// TODO: improve this
	if newPage.Title == "" {
		newPage.Title = page.Title
	}
	if newPage.Excerpt == "" {
		newPage.Excerpt = page.Excerpt
	}
	if newPage.Content == "" {
		newPage.Content = page.Content
	}
	if data.Published == nil {
		newPage.Published = page.Published
	}
	if newPage.PublishedAt == "" {
		newPage.PublishedAt = page.PublishedAt
	}

	// if not the original author, they can't change authors
	if author.UserId != page.Author.UserId && (data.Author != "" || data.Authors != nil) {
		render.Render(w, r, ErrForbidden(errors.New("you must be the original author in order to change authors")))
		return
	}

	if data.Author == "" {
		newPage.Author = page.Author
	} else {
		// The original author is making another author the original author.
		newOriginalAuthor, err := env.authors.Get(data.Author)
		if err != nil {
			render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", data.Author)))
			return
		}
		newPage.Author = newOriginalAuthor
	}

	if data.Authors == nil {
		newPage.Authors = page.Authors
	} else {
		authors, missingAuthorId, err := AuthorIdsToAuthors(data.Authors, env)
		if err != nil {
			if missingAuthorId != nil {
				render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
				return
			}
			render.Render(w, r, ErrInternal(err))
			panic(err)
		}

		newPage.Authors = authors
	}

	err := env.pages.Update(newPage)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	resp, err := NewPageResponse(newPage, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, resp)
}

func (env *Env) PageDelete(w http.ResponseWriter, r *http.Request) {
	page := r.Context().Value(pageCtxKey{}).(*models.Page)

	err := env.pages.Delete(page.Slug)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

type PageRequest struct {
	*models.Page
	Published *bool    `json:"published"`
	Author    string   `json:"author"`
	Authors   []string `json:"authors"`
}

func (pr *PageRequest) Bind(r *http.Request) error {
	if pr == nil {
		return errors.New("missing required Page fields")
	}

	if pr.Page.PublishedAt != "" {
	_, err := time.Parse(constants.PublishedAtFormat, pr.Page.PublishedAt)
	if err != nil {
		return fmt.Errorf("time must be in the format of %s", constants.PublishedAtFormat)
		}
	}

	if pr.Published != nil {
		pr.Page.Published = *pr.Published
	}

	return nil
}

type PageResponse struct {
	*models.Page
	Authors []*AuthorResponse `json:"authors"`
}

func (resp *PageResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func NewPageResponse(page *models.Page, env *Env) (*PageResponse, error) {
	resp := &PageResponse{Page: page}

	// Fetch page's authors
	authors, err := env.authors.OfPage(page.Slug)
	if err != nil {
		return nil, err
	}
	authorsResp := NewAuthorListResponse(authors)
	resp.Authors = authorsResp

	return resp, nil
}

func NewPageListResponse(pages []*models.Page, env *Env) ([]render.Renderer, error) {
	list := []render.Renderer{}
	for _, page := range pages {
		pageResp, err := NewPageResponse(page, env)
		if err != nil {
			return nil, err
		}
		list = append(list, pageResp)
	}
	return list, nil
}
