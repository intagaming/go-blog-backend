package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/go-sql-driver/mysql"
	"hxann.com/blog/api/auth"
	"hxann.com/blog/api/middleware"
	"hxann.com/blog/api/resp"
	"hxann.com/blog/constants"
	"hxann.com/blog/models"
)

type Posts struct {
	posts   *models.PostModel
	authors *models.AuthorModel
}

func (p *Posts) PostsGet(w http.ResponseWriter, r *http.Request) {
	// Fetch posts from db
	modelPosts, err := p.posts.All()

	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	postsResp, err := p.NewPostListResponse(modelPosts)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	render.RenderList(w, r, postsResp)
}

func (p *Posts) PostsPost(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.RequestAuthorCtxKey{}).(*models.Author)

	data := &PostRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, resp.ErrBadRequest(err))
		return
	}

	post := data.Post

	// Check required fields
	if post.Slug == "" || post.Title == "" || post.Excerpt == "" ||
		post.Content == "" {
		render.Render(w, r, resp.ErrBadRequest(errors.New("some of the required fields are not present. Required fields: slug, title, excerpt, content")))
		return
	}

	authorIds := append(data.Authors, author.UserId)
	authors, missingAuthorId, err := p.AuthorIdsToAuthors(authorIds)
	if err != nil {
		if missingAuthorId != nil {
			render.Render(w, r, resp.ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
			return
		}
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}
	for _, dbAuthor := range authors {
		if dbAuthor.UserId == author.UserId {
			post.Author = dbAuthor
		} else {
			post.Authors = append(post.Authors, dbAuthor)
		}
	}

	if err := p.posts.Add(post); err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == 1062 {
				render.Render(w, r, resp.ErrDuplicate(err))
				return
			}
		}
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	insertedPost, err := p.posts.Get(post.Slug)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
		panic(err)
	}

	resp, err := p.NewPostResponse(insertedPost)
	if err != nil {
		w.WriteHeader(http.StatusCreated)
		panic(err)
	}

	render.Status(r, http.StatusCreated)
	render.Render(w, r, resp)
}

func (p *Posts) PostGet(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(middleware.PostCtxKey{}).(*models.Post)

	postResp, err := p.NewPostResponse(post)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, postResp)
}

func (p *Posts) PostPut(w http.ResponseWriter, r *http.Request) {
	author := r.Context().Value(middleware.RequestAuthorCtxKey{}).(*models.Author)

	post := r.Context().Value(middleware.PostCtxKey{}).(*models.Post)

	data := &PostRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, resp.ErrBadRequest(err))
		return
	}

	newPost := data.Post
	if newPost == nil {
		newPost = &models.Post{}
	}
	// Provides the slug from context
	newPost.Slug = post.Slug
	// Fill in missing fields
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
	if !auth.IsAdmin(r) && author.UserId != post.Author.UserId && (data.Author != "" || data.Authors != nil) {
		render.Render(w, r, resp.ErrForbidden(errors.New("you must be the original author in order to change authors")))
		return
	}

	if data.Author == "" {
		newPost.Author = post.Author
	} else {
		// The original author is making another author the original author.
		newOriginalAuthor, err := p.authors.Get(data.Author)
		if err != nil {
			render.Render(w, r, resp.ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", data.Author)))
			return
		}
		newPost.Author = newOriginalAuthor
	}

	if data.Authors == nil {
		newPost.Authors = post.Authors
	} else {
		authors, missingAuthorId, err := p.AuthorIdsToAuthors(data.Authors)
		if err != nil {
			if missingAuthorId != nil {
				render.Render(w, r, resp.ErrNotFoundCustom(fmt.Errorf("couldn't find author with user_id of %s", *missingAuthorId)))
				return
			}
			render.Render(w, r, resp.ErrInternal(err))
			panic(err)
		}

		newPost.Authors = authors
	}

	err := p.posts.Update(newPost)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	postResp, err := p.NewPostResponse(newPost)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
		panic(err)
	}

	render.Render(w, r, postResp)
}

func (p *Posts) PostDelete(w http.ResponseWriter, r *http.Request) {
	post := r.Context().Value(middleware.PostCtxKey{}).(*models.Post)

	err := p.posts.Delete(post.Slug)
	if err != nil {
		render.Render(w, r, resp.ErrInternal(err))
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

	if pr.Post != nil && pr.Post.PublishedAt != "" {
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

	// last_post_slug and next_post_slug are nullable, semantically.
	LastPostSlug *string `json:"last_post_slug"`
	NextPostSlug *string `json:"next_post_slug"`
}

func (resp *PostResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (p *Posts) NewPostResponse(post *models.Post) (*PostResponse, error) {
	resp := &PostResponse{Post: post}

	// Fetch post's authors
	authors, err := p.authors.OfPost(post.Slug)
	if err != nil {
		return nil, err
	}
	authorsResp := NewAuthorListResponse(authors)
	resp.Authors = authorsResp

	// Fetch LastPostSlug & NextPostSlug
	last, next, err := p.posts.GetLastAndNextPostSlug(post.Slug)
	if err != nil {
		return nil, err
	}
	if last != "" {
		resp.LastPostSlug = &last
	}
	if next != "" {
		resp.NextPostSlug = &next
	}

	return resp, nil
}

func (p *Posts) NewPostListResponse(posts []*models.Post) ([]render.Renderer, error) {
	list := []render.Renderer{}
	for _, post := range posts {
		postResp, err := p.NewPostResponse(post)
		if err != nil {
			return nil, err
		}
		list = append(list, postResp)
	}
	return list, nil
}

// AuthorIdsToAuthors returns a list of Author from authorIds
func (p *Posts) AuthorIdsToAuthors(authorIds []string) (authors []*models.Author, missingAuthorId *string, err error) {
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
		author, err := p.authors.Get(authorId)
		if err != nil {
			return nil, &authorId, err
		}
		authors = append(authors, author)
	}

	return authors, nil, nil
}

func NewPosts(posts *models.PostModel, authors *models.AuthorModel) *Posts {
	return &Posts{
		posts:   posts,
		authors: authors,
	}
}
