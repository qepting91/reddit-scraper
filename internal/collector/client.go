package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/loganintech/go-reddit/v2/reddit"
	"github.com/qepting91/reddit-scraper/internal/domain"
	"golang.org/x/time/rate"
)

type RedditClient struct {
	client  *reddit.Client
	limiter *rate.Limiter
}

// NewClient now requires a userAgent string to comply with Reddit's API rules
func NewClient(id, secret, user, pass, userAgent string) (*RedditClient, error) {
	creds := reddit.Credentials{ID: id, Secret: secret, Username: user, Password: pass}

	// Use the passed userAgent instead of a hardcoded string
	client, err := reddit.NewClient(creds, reddit.WithUserAgent(userAgent))
	if err != nil {
		return nil, err
	}

	// Rate Limit: Token Bucket Algorithm
	// 100 requests / 10 mins = ~1 request every 600ms
	limiter := rate.NewLimiter(rate.Every(600*time.Millisecond), 1)

	return &RedditClient{client: client, limiter: limiter}, nil
}

func (rc *RedditClient) FetchNewPosts(ctx context.Context, sub string, limit int) ([]domain.Post, error) {
	// Wait for token
	if err := rc.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	posts, _, err := rc.client.Subreddit.NewPosts(ctx, sub, &reddit.ListOptions{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("API error: %w", err)
	}

	var result []domain.Post
	for _, p := range posts {
		result = append(result, domain.Post{
			ID:           p.ID,
			Title:        p.Title,
			Subreddit:    p.SubredditNamePrefixed,
			Author:       p.Author,
			URL:          p.URL,
			Score:        p.Score,
			CommentCount: p.NumberOfComments,
			CreatedUTC:   float64(p.Created.Time.Unix()),
		})
	}
	return result, nil
}
