package openapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hxann.com/blog/models"
)

func PopularizePostModel(env *Env, modelPost *models.Post) (*Post, error) {
	modelAuthors, err := env.authors.OfPost(modelPost.Slug)
	if err != nil {
		return nil, err
	}

	var authors []Author
	for _, modelAuthor := range modelAuthors {
		authors = append(authors, *PopularizeAuthor(&modelAuthor))
	}

	post := &Post{
		Slug:    modelPost.Slug,
		Title:   modelPost.Title,
		Excerpt: modelPost.Excerpt,
		Authors: authors,
	}

	if modelPost.PublishedAt.Valid {
		post.PublishedAt = &modelPost.PublishedAt.Time
	}

	// TODO: lastPostSlug, nextPostSlug

	return post, nil
}

func (env *Env) PostsAllGet(c *gin.Context) {
	// Fetch posts from db
	modelPosts, err := env.posts.All()

	if err != nil {
		ResponseWithError(c, http.StatusInternalServerError, &err)
		return
	}

	// Popularize posts
	var posts []Post
	for _, modelPost := range modelPosts {
		post, err := PopularizePostModel(env, &modelPost)
		if err != nil {
			ResponseWithError(c, http.StatusInternalServerError, &err)
			return
		}
		posts = append(posts, *post)
	}

	c.JSON(http.StatusOK, posts)
}
