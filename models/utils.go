package models

import (
	"time"

	"hxann.com/blog/blog/constants"
)

func CurrentTime() string {
	return time.Now().Format(constants.PublishedAtFormat)
}
