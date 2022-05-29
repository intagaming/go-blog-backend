package models

import (
	"database/sql"
	"errors"
	"time"

	"hxann.com/blog/blog/constants"
)

type Post struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Excerpt     string    `json:"excerpt"`
	Content     string    `json:"content"`
	Published   bool      `json:"published"`
	PublishedAt string    `json:"published_at,omitempty"`
	Author      *Author   `json:"author"`
	Authors     []*Author `json:"authors,omitempty"`
	CoverUrl    *string   `json:"cover_url"`
}

func (post *Post) IsAuthor(author *Author) bool {
	if post.Author.UserId == author.UserId {
		return true
	}
	for _, postAuthor := range post.Authors {
		if postAuthor.UserId == author.UserId {
			return true
		}
	}
	return false
}

type PostModel struct {
	DB *sql.DB
}

func (m PostModel) All() ([]*Post, error) {
	// Fetch all posts' information
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

	publications := make(map[string]string)
	for publishedRows.Next() {
		var slug string
		var timeBytes []byte
		err := publishedRows.Scan(&slug, &timeBytes)
		if err != nil {
			return nil, err
		}
		publications[slug] = string(timeBytes)
	}
	if err = publishedRows.Err(); err != nil {
		return nil, err
	}

	coverUrlRows, err := m.DB.Query(`
		SELECT post_slug, cover_url
		FROM posts_cover_url`)
	if err != nil {
		return nil, err
	}
	defer publishedRows.Close()

	coverUrls := make(map[string]string)
	for coverUrlRows.Next() {
		var slug string
		var coverUrl string
		err := coverUrlRows.Scan(&slug, &coverUrl)
		if err != nil {
			return nil, err
		}
		coverUrls[slug] = coverUrl
	}
	if err = coverUrlRows.Err(); err != nil {
		return nil, err
	}

	// Populating a list of posts
	var posts []*Post

	for rows.Next() {
		var post Post

		err := rows.Scan(&post.Slug, &post.Title, &post.Excerpt, &post.Content)
		if err != nil {
			return nil, err
		}

		if publishedAt, ok := publications[post.Slug]; ok {
			post.Published = true
			post.PublishedAt = publishedAt
		}

		m.FillAuthors(&post)

		if coverUrl, ok := coverUrls[post.Slug]; ok {
			post.CoverUrl = &coverUrl
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
		post.Published = true
		post.PublishedAt = string(timeBytes)
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	m.FillAuthors(&post)

	var coverUrl string
	err = m.DB.QueryRow(`
		SELECT cover_url
		FROM posts_cover_url
		WHERE post_slug = ?`, slug).Scan(&coverUrl)
	if err == nil {
		post.CoverUrl = &coverUrl
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	return &post, nil
}

func (m PostModel) Add(post *Post) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO posts
		(slug, title, excerpt, content)
		VALUES (?, ?, ?, ?)`, post.Slug, post.Title, post.Excerpt, post.Content)
	if err != nil {
		return err
	}

	if post.Published {
		if post.PublishedAt == "" {
			now := time.Now()
			post.PublishedAt = now.Format(constants.PublishedAtFormat)
		}
		_, err = tx.Exec(`
			INSERT INTO posts_publication
			(post_slug, published_at)
			VALUES (?, ?)`, post.Slug, post.PublishedAt)
		if err != nil {
			return err
		}
	}

	if post.CoverUrl != nil {
		_, err = tx.Exec(`
			INSERT INTO posts_cover_url
			(post_slug, cover_url)
			VALUES (?, ?)`, post.Slug, *post.CoverUrl)
		if err != nil {
			return err
		}
	}

	addAuthorStmt, err := tx.Prepare(`
		INSERT INTO posts_authors
		(post_slug, author_user_id, is_original)
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer addAuthorStmt.Close()

	for _, author := range post.Authors {
		if author.UserId == post.Author.UserId {
			continue
		}

		_, err := addAuthorStmt.Exec(post.Slug, author.UserId, 0)
		if err != nil {
			return err
		}
	}
	_, err = addAuthorStmt.Exec(post.Slug, post.Author.UserId, 1)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Update updates the post and also modifies newPost as the new post is in the
// database.
func (m PostModel) Update(newPost *Post) error {
	post, err := m.Get(newPost.Slug)
	if err != nil {
		return err
	}

	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE posts
		SET title=?, excerpt=?, content=?
		WHERE slug=?`, newPost.Title, newPost.Excerpt, newPost.Content, newPost.Slug)

	if err != nil {
		return err
	}

	if !post.Published && newPost.Published {
		if newPost.PublishedAt == "" {
			now := time.Now()
			newPost.PublishedAt = now.Format(constants.PublishedAtFormat)
		}
		_, err := tx.Exec(`
			INSERT INTO posts_publication
			(post_slug, published_at)
			VALUES (?, ?)`, newPost.Slug, newPost.PublishedAt)
		if err != nil {
			return err
		}
	} else if post.Published && !newPost.Published {
		_, err := tx.Exec(`DELETE FROM posts_publication WHERE post_slug=?`, newPost.Slug)
		if err != nil {
			return err
		}
		newPost.Published = false
		newPost.PublishedAt = ""
	} else if newPost.Published && newPost.PublishedAt != "" {
		_, err := tx.Exec(`
			UPDATE posts_publication
			SET published_at = ?
			WHERE post_slug = ?`, newPost.PublishedAt, newPost.Slug)
		if err != nil {
			return err
		}
	}

	if newPost.CoverUrl != nil {
		_, err = tx.Exec(`
			INSERT INTO posts_cover_url
			(post_slug, cover_url)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE cover_url = ?
			`, newPost.Slug, *newPost.CoverUrl, *newPost.CoverUrl)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`DELETE FROM posts_authors WHERE post_slug = ?`, newPost.Slug)
	if err != nil {
		return err
	}

	addAuthorStmt, err := tx.Prepare(`
		INSERT INTO posts_authors
		(post_slug, author_user_id, is_original)
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer addAuthorStmt.Close()

	for _, author := range newPost.Authors {
		if author.UserId == newPost.Author.UserId {
			continue
		}

		_, err := addAuthorStmt.Exec(newPost.Slug, author.UserId, 0)
		if err != nil {
			return err
		}
	}
	_, err = addAuthorStmt.Exec(newPost.Slug, newPost.Author.UserId, 1)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (m PostModel) Delete(slug string) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM posts WHERE slug = ?`, slug)
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

	_, err = tx.Exec(`DELETE FROM posts_publication WHERE post_slug = ?`, slug)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM posts_cover_url WHERE post_slug = ?`, slug)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM posts_authors WHERE post_slug = ?`, slug)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// FillAuthors fills in post.Author and post.Authors
func (m PostModel) FillAuthors(post *Post) error {
	rows, err := m.DB.Query(`
			SELECT user_id, full_name, email, bio, posts_authors.is_original
			FROM authors
			INNER JOIN posts_authors ON posts_authors.author_user_id = authors.user_id
			WHERE posts_authors.post_slug = ?
		`, post.Slug)
	if err != nil {
		return err
	}
	defer rows.Close()

	var authors []*Author
	for rows.Next() {
		var author Author
		var isOriginal bool
		err := rows.Scan(&author.UserId, &author.FullName, &author.Email, &author.Bio, &isOriginal)
		if err != nil {
			return err
		}
		authors = append(authors, &author)

		if isOriginal {
			post.Author = &author
		}
	}

	post.Authors = authors
	return nil
}

// TODO: might have non-repeatable read problem here, because in-between these 2
// queries, the data in the table might change.
func (m PostModel) GetLastAndNextPostSlug(slug string) (last string, next string, err error) {
	err = m.DB.QueryRow(`
		SELECT post_slug
		FROM posts_publication
		WHERE published_at < (
			SELECT published_at FROM posts_publication WHERE post_slug = ?
		)
		ORDER BY published_at DESC
		LIMIT 1`, slug).Scan(&last)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	err = m.DB.QueryRow(`
		SELECT post_slug
		FROM posts_publication
		WHERE published_at > (
			SELECT published_at FROM posts_publication WHERE post_slug = ?
		)
		ORDER BY published_at ASC
		LIMIT 1`, slug).Scan(&next)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	err = nil
	return
}
