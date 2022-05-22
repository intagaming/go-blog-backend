package openapi

import "github.com/gin-gonic/gin"

func ResponseWithError(c *gin.Context, code int, err *error) {
	c.JSON(code, &ErrorResponse{
		Status:  int32(code),
		Message: (*err).Error(),
	})
}
