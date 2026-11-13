package model

import "github.com/google/uuid"

type Poster struct {
	Id            uuid.UUID `json:"id"`
	Label         string    `json:"name"`
	Text          string    `json:"text"`
	Author        User      `json:"author"`
	CommentsCount int64     `json:"comments_count"`
	LikesCount    int64     `json:"likes_count"`
	PublicationAt string    `json:"publication_at"`
}

type User struct {
	Id        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url"`
}

type Comment struct {
	Id         uuid.UUID `json:"id"`
	Author     User      `json:"author"`
	LikesCount int64     `json:"likes_count"`
}
