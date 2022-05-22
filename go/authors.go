package openapi

import "hxann.com/blog/models"

func PopularizeAuthor(author *models.Author) *Author {
	return &Author{
		UserId:   author.UserId,
		FullName: author.FullName,
		Email:    author.Email,
		Bio:      author.Bio,
	}
}
