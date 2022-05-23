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

func (env *Env) PostsGet(w http.ResponseWriter, r *http.Request) {
	// Fetch posts from db
	modelPosts, err := env.posts.All()

	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	postsResp, err := NewPostListResponse(modelPosts, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.RenderList(w, r, postsResp)
}

func (env *Env) PostsPost(w http.ResponseWriter, r *http.Request) {
	data := &PostRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	post := data.Post

	// Check required fields
	if post.Slug == "" || post.Title == "" || post.Excerpt == "" ||
		post.Content == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("some of the required fields are not present. Required fields: slug, title, excerpt, content")))
		return
	}

	if err := env.posts.Add(post); err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	resp, err := NewPostResponse(post, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, resp)
}

type postCtxKey struct{}

func (env *Env) PostContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var post *models.Post
		var err error

		if slug := chi.URLParam(r, "slug"); slug != "" {
			post, err = env.posts.Get(slug)
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
		ctx := context.WithValue(r.Context(), postCtxKey{}, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (env *Env) PostGet(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(postCtxKey{}).(*models.Post)

	resp, err := NewPostResponse(post, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, resp)
}

type PostRequest struct {
	*models.Post
}

func (pr *PostRequest) Bind(r *http.Request) error {
	if pr.Post == nil {
		return errors.New("missing required Post fields")
	}

	return nil
}

type PostResponse struct {
	*models.Post
	PublishedAt string            `json:"published_at,omitempty"`
	Authors     []*AuthorResponse `json:"authors"`
	// TODO: coverUrl, lastPostSlug, nextPostSlug
}

func (resp *PostResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if resp.Post.PublishedAt != nil {
		resp.PublishedAt = resp.Post.PublishedAt.Format("2006-01-02 15:04:05")
	}

	return nil
}

func NewPostResponse(post *models.Post, env *Env) (*PostResponse, error) {
	resp := &PostResponse{Post: post}

	// Fetch post's authors
	authors, err := env.authors.OfPost(post.Slug)
	if err != nil {
		return nil, err
	}
	authorsResp := NewAuthorListResponse(authors)
	resp.Authors = authorsResp

	return resp, nil
}

func NewPostListResponse(posts []*models.Post, env *Env) ([]render.Renderer, error) {
	list := []render.Renderer{}
	for _, post := range posts {
		postResp, err := NewPostResponse(post, env)
		if err != nil {
			return nil, err
		}
		list = append(list, postResp)
	}
	return list, nil
}
