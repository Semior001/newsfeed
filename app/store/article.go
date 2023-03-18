// Package store contains models and interfaces for application.
package store

// Article is a struct that contains the extracted article.
type Article struct {
	Title        string `json:"title"`
	Excerpt      string `json:"excerpt"`
	Content      string `json:"content"`
	Author       string `json:"author"`
	ImageURL     string `json:"image_url"`
	BulletPoints string `json:"bullet_points"`
}
