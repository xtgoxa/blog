package main

import (
	"database/sql"
	"time"
)

// PostRepository handles database operations for posts
type PostRepository struct {
	db *sql.DB
}

// NewPostRepository creates a new PostRepository
func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// GetAll retrieves all posts
func (r *PostRepository) GetAll() ([]Post, error) {
	rows, err := r.db.Query(`
		SELECT id, title, content, author, category, tags, created_at, updated_at 
		FROM posts 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Author, &p.Category, &p.Tags,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.Format("2006-01-02")
		p.UpdatedAt = updatedAt.Format("2006-01-02")
		posts = append(posts, p)
	}
	return posts, nil
}

// GetByID retrieves a post by ID
func (r *PostRepository) GetByID(id int64) (*Post, error) {
	var p Post
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(`
		SELECT id, title, content, author, category, tags, created_at, updated_at 
		FROM posts 
		WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Title, &p.Content, &p.Author, &p.Category, &p.Tags,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = createdAt.Format("2006-01-02")
	p.UpdatedAt = updatedAt.Format("2006-01-02")
	return &p, nil
}

// Create creates a new post
func (r *PostRepository) Create(p *Post) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO posts (title, content, author, category, tags) 
		VALUES (?, ?, ?, ?, ?)
	`, p.Title, p.Content, p.Author, p.Category, p.Tags)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Update updates an existing post
func (r *PostRepository) Update(p *Post) error {
	_, err := r.db.Exec(`
		UPDATE posts 
		SET title = ?, content = ?, author = ?, category = ?, tags = ?, updated_at = NOW()
		WHERE id = ?
	`, p.Title, p.Content, p.Author, p.Category, p.Tags, p.ID)
	return err
}

// Delete deletes a post by ID
func (r *PostRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM posts WHERE id = ?", id)
	return err
}

// Search searches posts by query and category
func (r *PostRepository) Search(query, category, sort string) ([]Post, error) {
	var args []interface{}
	sql := `
		SELECT id, title, content, author, category, tags, created_at, updated_at 
		FROM posts 
		WHERE 1=1
	`

	if query != "" {
		sql += " AND (title LIKE ? OR content LIKE ? OR tags LIKE ?)"
		searchTerm := "%" + query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if category != "" {
		sql += " AND category = ?"
		args = append(args, category)
	}

	if sort == "oldest" {
		sql += " ORDER BY created_at ASC"
	} else {
		sql += " ORDER BY created_at DESC"
	}

	rows, err := r.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var createdAt, updatedAt time.Time
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Author, &p.Category, &p.Tags,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.Format("2006-01-02")
		p.UpdatedAt = updatedAt.Format("2006-01-02")
		posts = append(posts, p)
	}
	return posts, nil
}

// Count returns total number of posts
func (r *PostRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
