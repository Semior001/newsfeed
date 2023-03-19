// Package store contains entities and services to process and contain them.
package store

import (
	"context"
	"errors"
)

// ErrNotFound is an error that is returned when the requested entity is not found.
var ErrNotFound = errors.New("not found")

// Interface defines methods for store
type Interface interface {
	Put(ctx context.Context, u User) error
	Get(ctx context.Context, chatID string) (User, error)
	List(ctx context.Context, req ListRequest) ([]User, error)
	Delete(ctx context.Context, chatID string) error
}

// ListRequest defines parameters for listing users from store.
type ListRequest struct{}

// Article is a struct that contains the extracted article.
type Article struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Excerpt      string `json:"excerpt"`
	Content      string `json:"content"`
	Author       string `json:"author"`
	ImageURL     string `json:"image_url"`
	BulletPoints string `json:"bullet_points"`
}

// User is a struct that contains the user's data.
type User struct {
	ChatID     string `json:"chat_id"`
	Username   string `json:"username"`
	Authorized bool   `json:"authorized"`
	Subscribed bool   `json:"subscribed"`
}
