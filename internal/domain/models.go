package domain

import "context"

// Target represents a scraping task
type Target struct {
	Subreddit string
	MinScore  int
}

// Post is the clean data structure for storage
type Post struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Subreddit    string   `json:"subreddit"`
	Author       string   `json:"author"`
	URL          string   `json:"url"`
	Score        int      `json:"score"`
	CommentCount int      `json:"comment_count"`
	CreatedUTC   float64  `json:"created_utc"`
	KeywordsHit  []string `json:"keywords_hit,omitempty"`
}

// Collector defines the interface for data fetching
type Collector interface {
	FetchNewPosts(ctx context.Context, subreddit string, limit int) ([]Post, error)
}
