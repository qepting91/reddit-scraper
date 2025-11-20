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
	time.Sleep(200 * time.Millisecond)

	var posts []domain.Post
	// We use keywords that exist in your input/keywords.csv to ensure the dashboard populates
	fakeKeywords := []string{"Mandiant", "CrowdStrike", "MISP", "Analyst1", "Recorded Future", "ZeroFox", "OpenCTI"}

	for i := 0; i < limit; i++ {
		// Randomly select a keyword to inject
		kw := fakeKeywords[rand.Intn(len(fakeKeywords))]

		posts = append(posts, domain.Post{
			ID:           fmt.Sprintf("mock_%s_%d", sub, i),
			Title:        fmt.Sprintf("[%s] New analysis regarding %s detected in sector", sub, kw),
			Subreddit:    sub, // Note: Removed "r/" prefix here to match typical API return or keep consistency
			Author:       "simulated_user",
			URL:          "http://localhost/mock-url",
			Score:        rand.Intn(500) + 5, // Ensure it meets min_score (usually 5 or 10)
			CommentCount: rand.Intn(50),
			CreatedUTC:   float64(time.Now().Unix()),
		})
	}
	return posts, nil
}
