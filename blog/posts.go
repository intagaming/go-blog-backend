package blog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-sql-driver/mysql"
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
	author := r.Context().Value(AuthorCtxKey{}).(*models.Author)

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

	authorIds := append(data.Authors, author.UserId)
	authorIds, missingAuthorId, err := ValidateAuthorIds(authorIds, env)
	if err != nil {
		if missingAuthorId != nil {
			render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
			return
		}
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	if err := env.posts.Add(post, authorIds); err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == 1062 {
				render.Render(w, r, ErrDuplicate(err))
				return
			}
		}
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	insertedPost, err := env.posts.Get(post.Slug)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
		panic(err)
	}

	resp, err := NewPostResponse(insertedPost, env)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
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

func (env *Env) PostPut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(AuthorCtxKey{}).(*models.Author)

	post := r.Context().Value(postCtxKey{}).(*models.Post)

	data := &PostRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	newPost := data.Post
	// Provides the slug from context
	newPost.Slug = post.Slug
	// Fill in missing fields
	// TODO: improve this
	if newPost.Title == "" {
		newPost.Title = post.Title
	}
	if newPost.Excerpt == "" {
		newPost.Excerpt = post.Excerpt
	}
	if newPost.Content == "" {
		newPost.Content = post.Content
	}
	if data.Published == nil {
		newPost.Published = post.Published
	}
	if newPost.PublishedAt == "" {
		newPost.PublishedAt = post.PublishedAt
	}

	err := env.posts.Update(newPost)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	if data.Authors != nil {
		authorIds := append(data.Authors, author.UserId)
		authorIds, missingAuthorId, err := ValidateAuthorIds(authorIds, env)
		if err != nil {
			if missingAuthorId != nil {
				render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
				return
			}
			render.Render(w, r, ErrInternal(err))
			panic(err)
		}
		env.posts.UpdateAuthors(post.Slug, authorIds)
	}

	resp, err := NewPostResponse(newPost, env)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, resp)
}

func (env *Env) PostDelete(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(postCtxKey{}).(*models.Post)

	err := env.posts.Delete(post.Slug)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

type PostRequest struct {
	*models.Post
	Published *bool    `json:"published"`
	Authors   []string `json:"authors"`
}

func (pr *PostRequest) Bind(r *http.Request) error {
	if pr == nil {
		return errors.New("missing required Post fields")
	}
	if pr.Published != nil {
		pr.Post.Published = *pr.Published
	}

	return nil
}

type PostResponse struct {
	*models.Post
	Authors []*AuthorResponse `json:"authors"`
	// TODO: coverUrl, lastPostSlug, nextPostSlug
}

func (resp *PostResponse) Render(w http.ResponseWriter, r *http.Request) error {
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

// ValidateAuthorIds makes sure all authorIds are valid authors, and modifies
// authorIds to only include any author once.
func ValidateAuthorIds(authorIds []string, env *Env) (uniqueAuthorIds []string, missingAuthorId *string, err error) {
	var authorIdsSet map[string]struct{} = make(map[string]struct{})
	for _, authorId := range authorIds {
		authorIdsSet[authorId] = struct{}{}
	}

	i := 0
	for k := range authorIdsSet {
		authorIds[i] = k
		i++
	}
	authorIds = authorIds[:i]

	// Assure that all of the authors in the request are valid
	for _, authorId := range authorIds {
		_, err := env.authors.Get(authorId)
		if err != nil {
			return nil, &authorId, err
		}
	}

	return authorIds, nil, nil
}
