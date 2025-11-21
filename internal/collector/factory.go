package collector

import (
	"fmt"
	"os"

	"github.com/qepting91/reddit-scraper/internal/domain"
)

// NewCollector selects the correct implementation based on the MODE
func NewCollector() (domain.Collector, error) {
	mode := os.Getenv("COLLECTOR_MODE")
	userAgent := os.Getenv("REDDIT_USER_AGENT")

	switch mode {
	case "api":
		return NewAPIClient(
			os.Getenv("REDDIT_CLIENT_ID"),
			os.Getenv("REDDIT_CLIENT_SECRET"),
			os.Getenv("REDDIT_USERNAME"),
			os.Getenv("REDDIT_PASSWORD"),
			userAgent,
		)
	case "public":
		if userAgent == "" {
			return nil, fmt.Errorf("REDDIT_USER_AGENT is required for public mode")
		}
		return NewPublicClient(userAgent)
	case "mock":
		return NewMockClient(), nil
	default:
		return nil, fmt.Errorf("unknown COLLECTOR_MODE: %s (use 'api', 'public', or 'mock')", mode)
	}
}
