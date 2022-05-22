package openapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (env *Env) PostsAllGet(c *gin.Context) {
	posts, err := env.posts.All()

	if err != nil {
		ResponseWithError(c, http.StatusInternalServerError, &err)
		return
	}

	c.JSON(http.StatusOK, posts)
}
