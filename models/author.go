package models

import (
	"database/sql"
)

type Author struct {
	UserId   string
	FullName string
	Email    string
	Bio      string
}

type AuthorModel struct {
	DB *sql.DB
}

func (m AuthorModel) OfPost(postSlug string) ([]Author, error) {
	rows, err := m.DB.Query(`
		SELECT user_id, full_name, email, bio
		FROM authors
		INNER JOIN posts_authors ON posts_authors.author_user_id = authors.user_id
		WHERE posts_authors.post_slug = ?
	`, postSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author

	for rows.Next() {
		var author Author

		err := rows.Scan(&author.UserId, &author.FullName, &author.Email, &author.Bio)
		if err != nil {
			return nil, err
		}

		authors = append(authors, author)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}
