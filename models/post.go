package models

import (
	"database/sql"
)

type Post struct {
	Slug        string
	Title       string
	Excerpt     string
	Content     string
	PublishedAt sql.NullTime
}

type PostModel struct {
	DB *sql.DB
}

func (m PostModel) All() ([]Post, error) {
	rows, err := m.DB.Query(`
		SELECT slug, title, excerpt, content, published_at
		FROM posts
		LEFT JOIN posts_publication ON posts_publication.post_slug = posts.slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var post Post

		err := rows.Scan(&post.Slug, &post.Title, &post.Excerpt, &post.Content, &post.PublishedAt)
		if err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
