package main

import (
	"database/sql"
	"time"
)

// Carousel represents a carousel slide
type Carousel struct {
	ID         int64
	Title      string
	Subtitle   string
	ButtonText string
	ButtonLink string
	ImageURL   string
	SortOrder  int
	IsActive   bool
	CreatedAt  string
}

// CarouselRepository handles database operations for carousel
type CarouselRepository struct {
	db *sql.DB
}

// NewCarouselRepository creates a new CarouselRepository
func NewCarouselRepository(db *sql.DB) *CarouselRepository {
	return &CarouselRepository{db: db}
}

// GetAll retrieves all carousel slides
func (r *CarouselRepository) GetAll() ([]Carousel, error) {
	rows, err := r.db.Query(`
		SELECT id, title, subtitle, button_text, button_link, image_url, sort_order, is_active, created_at 
		FROM carousel 
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slides []Carousel
	for rows.Next() {
		var s Carousel
		var createdAt time.Time
		var subtitle, buttonText, buttonLink, imageURL sql.NullString
		err := rows.Scan(
			&s.ID, &s.Title, &subtitle, &buttonText, &buttonLink, &imageURL,
			&s.SortOrder, &s.IsActive, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		s.Subtitle = subtitle.String
		s.ButtonText = buttonText.String
		s.ButtonLink = buttonLink.String
		s.ImageURL = imageURL.String
		s.CreatedAt = createdAt.Format("2006-01-02")
		slides = append(slides, s)
	}
	return slides, nil
}

// GetActive retrieves only active carousel slides
func (r *CarouselRepository) GetActive() ([]Carousel, error) {
	rows, err := r.db.Query(`
		SELECT id, title, subtitle, button_text, button_link, image_url, sort_order, is_active, created_at 
		FROM carousel 
		WHERE is_active = TRUE
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slides []Carousel
	for rows.Next() {
		var s Carousel
		var createdAt time.Time
		var subtitle, buttonText, buttonLink, imageURL sql.NullString
		err := rows.Scan(
			&s.ID, &s.Title, &subtitle, &buttonText, &buttonLink, &imageURL,
			&s.SortOrder, &s.IsActive, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		s.Subtitle = subtitle.String
		s.ButtonText = buttonText.String
		s.ButtonLink = buttonLink.String
		s.ImageURL = imageURL.String
		s.CreatedAt = createdAt.Format("2006-01-02")
		slides = append(slides, s)
	}
	return slides, nil
}

// GetByID retrieves a carousel slide by ID
func (r *CarouselRepository) GetByID(id int64) (*Carousel, error) {
	var s Carousel
	var createdAt time.Time
	var subtitle, buttonText, buttonLink, imageURL sql.NullString
	err := r.db.QueryRow(`
		SELECT id, title, subtitle, button_text, button_link, image_url, sort_order, is_active, created_at 
		FROM carousel 
		WHERE id = ?
	`, id).Scan(
		&s.ID, &s.Title, &subtitle, &buttonText, &buttonLink, &imageURL,
		&s.SortOrder, &s.IsActive, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	s.Subtitle = subtitle.String
	s.ButtonText = buttonText.String
	s.ButtonLink = buttonLink.String
	s.ImageURL = imageURL.String
	s.CreatedAt = createdAt.Format("2006-01-02")
	return &s, nil
}

// Create creates a new carousel slide
func (r *CarouselRepository) Create(s *Carousel) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO carousel (title, subtitle, button_text, button_link, image_url, sort_order, is_active) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, s.Title, s.Subtitle, s.ButtonText, s.ButtonLink, s.ImageURL, s.SortOrder, s.IsActive)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Update updates an existing carousel slide
func (r *CarouselRepository) Update(s *Carousel) error {
	_, err := r.db.Exec(`
		UPDATE carousel 
		SET title = ?, subtitle = ?, button_text = ?, button_link = ?, image_url = ?, sort_order = ?, is_active = ?
		WHERE id = ?
	`, s.Title, s.Subtitle, s.ButtonText, s.ButtonLink, s.ImageURL, s.SortOrder, s.IsActive, s.ID)
	return err
}

// Delete deletes a carousel slide by ID
func (r *CarouselRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM carousel WHERE id = ?", id)
	return err
}

// ToggleActive toggles the active status of a carousel slide
func (r *CarouselRepository) ToggleActive(id int64, active bool) error {
	_, err := r.db.Exec("UPDATE carousel SET is_active = ? WHERE id = ?", active, id)
	return err
}
