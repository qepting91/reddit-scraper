package collector

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/qepting91/reddit-scraper/internal/domain"
)

// MockClient implements domain.Collector but returns fake data
type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (mc *MockClient) FetchNewPosts(ctx context.Context, sub string, limit int) ([]domain.Post, error) {
	// Simulate network latency (nice for testing concurrency)
	time.Sleep(500 * time.Millisecond)

	var posts []domain.Post
	for i := 0; i < limit; i++ {
		// Generate random "fake" posts
		posts = append(posts, domain.Post{
			ID:           fmt.Sprintf("mock_%s_%d", sub, i),
			Title:        fmt.Sprintf("[%s] Simulated Threat Intel Report #%d: Zero-Day Found", sub, i),
			Subreddit:    "r/" + sub,
			Author:       "simulated_user",
			URL:          "http://localhost/mock-url",
			Score:        rand.Intn(500),
			CommentCount: rand.Intn(50),
			CreatedUTC:   float64(time.Now().Unix()),
		})
	}
	return posts, nil
}
