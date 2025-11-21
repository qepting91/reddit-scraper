package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/loganintech/go-reddit/v2/reddit"
	"github.com/qepting91/reddit-scraper/internal/domain"
	"golang.org/x/time/rate"
)

type APIClient struct {
	client  *reddit.Client
	limiter *rate.Limiter
}

func NewAPIClient(id, secret, user, pass, userAgent string) (*APIClient, error) {
	creds := reddit.Credentials{ID: id, Secret: secret, Username: user, Password: pass}

	client, err := reddit.NewClient(creds, reddit.WithUserAgent(userAgent))
	if err != nil {
		return nil, err
	}

	// API Rate Limit: ~60 reqs/min (safe buffer)
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

	return &APIClient{client: client, limiter: limiter}, nil
}

func (ac *APIClient) FetchNewPosts(ctx context.Context, sub string, limit int) ([]domain.Post, error) {
	if err := ac.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	posts, _, err := ac.client.Subreddit.NewPosts(ctx, sub, &reddit.ListOptions{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("authenticated api error: %w", err)
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
