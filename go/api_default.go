/*
 * Go Blog API
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NavsGet - Returns an array of pages on the navigation bar.
func NavsGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// NavsPut - Modify the navigation bar's pages.
func NavsPut(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PagesAllGet - Returns all pages
func PagesAllGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PagesSlugDelete - Delete a page
func PagesSlugDelete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PagesSlugGet - Returns a page's details
func PagesSlugGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PagesSlugPost - Create a page
func PagesSlugPost(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PagesSlugPut - Edit a page
func PagesSlugPut(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PostsGet - Returns a list of blog posts
func PostsGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PostsSlugDelete - Delete a post
func PostsSlugDelete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PostsSlugGet - Returns a post's details
func PostsSlugGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PostsSlugPost - Create a post
func PostsSlugPost(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// PostsSlugPut - Edit a post
func PostsSlugPut(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
