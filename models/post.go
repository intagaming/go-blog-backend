package models

import (
	"database/sql"
	"errors"
	"time"
)

type Post struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     string     `json:"excerpt"`
	Content     string     `json:"content"`
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"-"`
}

// type publishedAt []byte

// func (pa publishedAt) Time() (time.Time, error) {
// 	return time.Parse("2006-01-02 15:04:05", string(pa))
// }

type PostModel struct {
	DB *sql.DB
}

func (m PostModel) All() ([]*Post, error) {
	rows, err := m.DB.Query(`
		SELECT slug, title, excerpt, content
		FROM posts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	publishedRows, err := m.DB.Query(`
		SELECT post_slug, published_at
		FROM posts_publication`)
	if err != nil {
		return nil, err
	}
	defer publishedRows.Close()

	publications := make(map[string]time.Time)
	for publishedRows.Next() {
		var slug string
		var timeBytes []byte
		err := publishedRows.Scan(&slug, &timeBytes)
		if err != nil {
			return nil, err
		}
		parsedTime, err := time.Parse("2006-01-02 15:04:05", string(timeBytes))
		if err != nil {
			return nil, err
		}
		publications[slug] = parsedTime
	}
	if err = publishedRows.Err(); err != nil {
		return nil, err
	}

	var posts []*Post

	for rows.Next() {
		var post Post

		err := rows.Scan(&post.Slug, &post.Title, &post.Excerpt, &post.Content)
		if err != nil {
			return nil, err
		}

		if publishedAt, ok := publications[post.Slug]; ok {
			post.Published = true
			post.PublishedAt = &publishedAt
		}

		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (m PostModel) Get(slug string) (*Post, error) {
	var post Post = Post{Slug: slug}

	err := m.DB.QueryRow(`
		SELECT title, excerpt, content
		FROM posts
		WHERE slug = ?`, slug).Scan(&post.Title, &post.Excerpt, &post.Content)
	if err != nil {
		return nil, err
	}

	var timeBytes []byte

	err = m.DB.QueryRow(`
		SELECT published_at
		FROM posts_publication
		WHERE post_slug = ?`, slug).Scan(&timeBytes)
	if err == nil {
		parsedTime, err := time.Parse("2006-01-02 15:04:05", string(timeBytes))
		if err != nil {
			return nil, err
		}
		post.Published = true
		post.PublishedAt = &parsedTime
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	return &post, nil
}

func (m PostModel) Add(post *Post) error {
	_, err := m.DB.Exec(`
		INSERT INTO posts
		(slug, title, excerpt, content)
		VALUES (?, ?, ?, ?)`, post.Slug, post.Title, post.Excerpt, post.Content)
	if err != nil {
		return err
	}
	return nil
}

func (m PostModel) Delete(slug string) error {
	res, err := m.DB.Exec(`DELETE FROM posts WHERE slug = ?`, slug)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no post were deleted")
	}

	return nil
}
