package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/qepting91/reddit-scraper/internal/domain"
	"golang.org/x/time/rate"
)

type PublicClient struct {
	httpClient *http.Client
	limiter    *rate.Limiter
	userAgent  string
}

type redditJSONResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID          string  `json:"id"`
				Title       string  `json:"title"`
				Subreddit   string  `json:"subreddit_name_prefixed"`
				Author      string  `json:"author"`
				URL         string  `json:"url"`
				Score       int     `json:"score"`
				NumComments int     `json:"num_comments"`
				CreatedUTC  float64 `json:"created_utc"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func NewPublicClient(userAgent string) (*PublicClient, error) {
	return &PublicClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		// Public JSON Limit: 1 req / 2 seconds (Stricter)
		limiter:   rate.NewLimiter(rate.Every(2*time.Second), 1),
		userAgent: userAgent,
	}, nil
}

func (pc *PublicClient) FetchNewPosts(ctx context.Context, sub string, limit int) ([]domain.Post, error) {
	if err := pc.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", sub, limit)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", pc.userAgent)

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("reddit public access status: %d", resp.StatusCode)
	}

	var rResp redditJSONResponse
	if err := json.NewDecoder(resp.Body).Decode(&rResp); err != nil {
		return nil, err
	}

	var posts []domain.Post
	for _, child := range rResp.Data.Children {
		d := child.Data
		posts = append(posts, domain.Post{
			ID:           d.ID,
			Title:        d.Title,
			Subreddit:    d.Subreddit,
			Author:       d.Author,
			URL:          d.URL,
			Score:        d.Score,
			CommentCount: d.NumComments,
			CreatedUTC:   d.CreatedUTC,
		})
	}
	return posts, nil
}
