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
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

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
			post.Author = dbAuthor
		} else {
			post.Authors = append(post.Authors, dbAuthor)
		}
	}

	if err := env.posts.Add(post); err != nil {
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
	author := r.Context().Value(authorCtxKey{}).(*models.Author)

	post := r.Context().Value(postCtxKey{}).(*models.Post)

	data := &PostRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	newPost := data.Post
	if newPost == nil {
		newPost = &models.Post{}
	}
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

	// if not the original author or blog's admin, they can't change authors
	if !IsAdmin(r) && author.UserId != post.Author.UserId && (data.Author != "" || data.Authors != nil) {
		render.Render(w, r, ErrForbidden(errors.New("you must be the original author in order to change authors")))
		return
	}

	if data.Author == "" {
		newPost.Author = post.Author
	} else {
		// The original author is making another author the original author.
		newOriginalAuthor, err := env.authors.Get(data.Author)
		if err != nil {
			render.Render(w, r, ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", data.Author)))
			return
		}
		newPost.Author = newOriginalAuthor
	}

	if data.Authors == nil {
		newPost.Authors = post.Authors
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

		newPost.Authors = authors
	}

	err := env.posts.Update(newPost)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		panic(err)
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
	Author    string   `json:"author"`
	Authors   []string `json:"authors"`
}

func (pr *PostRequest) Bind(r *http.Request) error {
	if pr == nil {
		return errors.New("missing required Post fields")
	}

	if pr.Post.PublishedAt != "" {
		_, err := time.Parse(constants.PublishedAtFormat, pr.Post.PublishedAt)
		if err != nil {
			return fmt.Errorf("time must be in the format of %s", constants.PublishedAtFormat)
		}
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

// AuthorIdsToAuthors returns a list of Author from authorIds
func AuthorIdsToAuthors(authorIds []string, env *Env) (authors []*models.Author, missingAuthorId *string, err error) {
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
		author, err := env.authors.Get(authorId)
		if err != nil {
			return nil, &authorId, err
		}
		authors = append(authors, author)
	}

	return authors, nil, nil
}
