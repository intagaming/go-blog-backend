package models

import (
	"time"

	"hxann.com/blog/constants"
)

func CurrentTime() string {
	return time.Now().Format(constants.PublishedAtFormat)
}
