package models

import (
	"database/sql"
	"errors"
	"time"
)

type Page struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Excerpt     string    `json:"excerpt"`
	Content     string    `json:"content"`
	Published   bool      `json:"published"`
	PublishedAt string    `json:"published_at,omitempty"`
	Author      *Author   `json:"author"`
	Authors     []*Author `json:"authors,omitempty"`
}

func (page *Page) IsAuthor(author *Author) bool {
	if page.Author.UserId == author.UserId {
		return true
	}
	for _, pageAuthor := range page.Authors {
		if pageAuthor.UserId == author.UserId {
			return true
		}
	}
	return false
}

type PageModel struct {
	DB *sql.DB
}

func (m PageModel) All() ([]*Page, error) {
	rows, err := m.DB.Query(`
		SELECT slug, title, excerpt, content
		FROM pages`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	publishedRows, err := m.DB.Query(`
		SELECT page_slug, published_at
		FROM pages_publication`)
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

	var pages []*Page

	for rows.Next() {
		var page Page

		err := rows.Scan(&page.Slug, &page.Title, &page.Excerpt, &page.Content)
		if err != nil {
			return nil, err
		}

		if publishedAt, ok := publications[page.Slug]; ok {
			page.Published = true
			page.PublishedAt = publishedAt
		}

		m.FillAuthors(&page)

		pages = append(pages, &page)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return pages, nil
}

func (m PageModel) Get(slug string) (*Page, error) {
	var page Page = Page{Slug: slug}

	err := m.DB.QueryRow(`
		SELECT title, excerpt, content
		FROM pages
		WHERE slug = ?`, slug).Scan(&page.Title, &page.Excerpt, &page.Content)
	if err != nil {
		return nil, err
	}

	var timeBytes []byte

	err = m.DB.QueryRow(`
		SELECT published_at
		FROM pages_publication
		WHERE page_slug = ?`, slug).Scan(&timeBytes)
	if err == nil {
		page.Published = true
		page.PublishedAt = string(timeBytes)
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	m.FillAuthors(&page)

	return &page, nil
}

func (m PageModel) Add(page *Page) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO pages
		(slug, title, excerpt, content)
		VALUES (?, ?, ?, ?)`, page.Slug, page.Title, page.Excerpt, page.Content)
	if err != nil {
		return err
	}

	if page.Published {
		if page.PublishedAt == "" {
			now := time.Now()
			page.PublishedAt = now.Format("2006-01-02 15:04:05")
		}
		_, err = tx.Exec(`
			INSERT INTO pages_publication
			(page_slug, published_at)
			VALUES (?, ?)`, page.Slug, page.PublishedAt)
	}
	if err != nil {
		return err
	}

	addAuthorStmt, err := tx.Prepare(`
		INSERT INTO pages_authors
		(page_slug, author_user_id, is_original)
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, author := range page.Authors {
		if author.UserId == page.Author.UserId {
			continue
		}

		_, err := addAuthorStmt.Exec(page.Slug, author.UserId, 0)
		if err != nil {
			return err
		}
	}
	_, err = addAuthorStmt.Exec(page.Slug, page.Author.UserId, 1)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Update updates the page and also modifies newPage as the new page is in the
// database.
func (m PageModel) Update(newPage *Page) error {
	page, err := m.Get(newPage.Slug)
	if err != nil {
		return err
	}

	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE pages
		SET title=?, excerpt=?, content=?
		WHERE slug=?`, newPage.Title, newPage.Excerpt, newPage.Content, newPage.Slug)

	if err != nil {
		return err
	}

	if !page.Published && newPage.Published {
		if newPage.PublishedAt == "" {
			now := time.Now()
			newPage.PublishedAt = now.Format("2006-01-02 15:04:05")
		}
		_, err := tx.Exec(`
			INSERT INTO pages_publication
			(page_slug, published_at)
			VALUES (?, ?)`, newPage.Slug, newPage.PublishedAt)
		if err != nil {
			return err
		}
	} else if page.Published && !newPage.Published {
		_, err := tx.Exec(`DELETE FROM pages_publication WHERE page_slug=?`, newPage.Slug)
		if err != nil {
			return err
		}
		newPage.Published = false
		newPage.PublishedAt = ""
	} else if newPage.Published && newPage.PublishedAt != "" {
		_, err := tx.Exec(`
			UPDATE pages_publication
			SET published_at = ?
			WHERE page_slug = ?`, newPage.PublishedAt, newPage.Slug)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`DELETE FROM pages_authors WHERE page_slug = ?`, newPage.Slug)
	if err != nil {
		return err
	}

	addAuthorStmt, err := tx.Prepare(`
		INSERT INTO pages_authors
		(page_slug, author_user_id, is_original)
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, author := range newPage.Authors {
		if author.UserId == newPage.Author.UserId {
			continue
		}

		_, err := addAuthorStmt.Exec(newPage.Slug, author.UserId, 0)
		if err != nil {
			return err
		}
	}
	_, err = addAuthorStmt.Exec(newPage.Slug, newPage.Author.UserId, 1)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (m PageModel) Delete(slug string) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM pages WHERE slug = ?`, slug)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no page were deleted")
	}

	_, err = tx.Exec(`DELETE FROM pages_publication WHERE page_slug = ?`, slug)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM pages_authors WHERE page_slug = ?`, slug)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// FillAuthors fills in page.Author and page.Authors
func (m PageModel) FillAuthors(page *Page) error {
	rows, err := m.DB.Query(`
			SELECT user_id, full_name, email, bio, pages_authors.is_original
			FROM authors
			INNER JOIN pages_authors ON pages_authors.author_user_id = authors.user_id
			WHERE pages_authors.page_slug = ?
		`, page.Slug)
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
			page.Author = &author
		}
	}

	page.Authors = authors
	return nil
}
