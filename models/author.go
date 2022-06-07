package models

import (
	"database/sql"
)

type Author struct {
	UserId   string `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
}

type AuthorModel struct {
	DB *sql.DB
}

func (m AuthorModel) CoAuthorsOfPost(postSlug string) ([]*Author, error) {
	rows, err := m.DB.Query(`
		SELECT user_id, full_name, email, bio
		FROM authors
		INNER JOIN posts_authors ON posts_authors.author_user_id = authors.user_id
		WHERE posts_authors.post_slug = ? AND posts_authors.is_original = 0
	`, postSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []*Author

	for rows.Next() {
		var author Author

		err := rows.Scan(&author.UserId, &author.FullName, &author.Email, &author.Bio)
		if err != nil {
			return nil, err
		}

		authors = append(authors, &author)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}

func (m AuthorModel) Get(userId string) (*Author, error) {
	var author Author = Author{UserId: userId}

	err := m.DB.QueryRow(`
		SELECT full_name, email, bio
		FROM authors
		WHERE user_id = ?`, userId).Scan(&author.FullName, &author.Email, &author.Bio)

	if err != nil {
		return nil, err
	}

	return &author, nil
}

func (m AuthorModel) Add(author *Author) error {
	_, err := m.DB.Exec(`
		INSERT INTO authors
		(user_id, full_name, email, bio)
		VALUES (?, ?, ?, ?)`, author.UserId, author.FullName, author.Email, author.Bio)
	if err != nil {
		return err
	}

	return nil
}

func (m AuthorModel) Update(newAuthor *Author) error {
	_, err := m.DB.Exec(`
		UPDATE authors
		SET full_name=?, email=?, bio=?
		WHERE user_id=?`, newAuthor.FullName, newAuthor.Email, newAuthor.Bio, newAuthor.UserId)
	if err != nil {
		return err
	}

	return nil
}
