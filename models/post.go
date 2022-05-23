package models

import (
	"database/sql"
	"time"
)

type Post struct {
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Excerpt     string      `json:"excerpt"`
	Content     string      `json:"content"`
	PublishedAt publishedAt `json:"-"`
}

type publishedAt []byte

func (pa publishedAt) Time() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", string(pa))
}

type PostModel struct {
	DB *sql.DB
}

func (m PostModel) All() ([]*Post, error) {
	rows, err := m.DB.Query(`
		SELECT slug, title, excerpt, content, published_at
		FROM posts
		LEFT JOIN posts_publication ON posts_publication.post_slug = posts.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post

	for rows.Next() {
		var post Post

		err := rows.Scan(&post.Slug, &post.Title, &post.Excerpt, &post.Content, &post.PublishedAt)
		if err != nil {
			return nil, err
		}

		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
