#!/usr/bin/env bash

# ==============================================================================
# Reddit Scraper Setup Script
# Author: github.com/qepting91
# Description: Scaffolds a secure, production-ready Go environment.
# ==============================================================================

# ------------------------------------------------------------------------------
# 1. Configuration & Safety
# ------------------------------------------------------------------------------
set -euo pipefail  # Fail on error, unset vars, and pipe failures

# ANSI Colors for Professional Output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROJECT_NAME="reddit-scraper"
MODULE_NAME="github.com/qepting91/reddit-scraper"
GO_VERSION="1.23" # Min version recommendation

# ------------------------------------------------------------------------------
# 2. Helper Functions
# ------------------------------------------------------------------------------
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is not installed. Please install it to proceed."
    fi
}

create_file() {
    local filepath="$1"
    local content="$2"
    mkdir -p "$(dirname "$filepath")"
    echo "$content" > "$filepath"
    log_info "Created: $filepath"
}

# ------------------------------------------------------------------------------
# 3. Pre-flight Checks
# ------------------------------------------------------------------------------
log_info "Starting pre-flight checks..."
check_dependency "go"
check_dependency "git"

# Verify Go version matches recommendation
current_go_ver=$(go version | { read -r _ _ v _; echo "${v#go}"; })
log_info "Found Go version: $current_go_ver (Recommended: $GO_VERSION+)"

if [ -d "$PROJECT_NAME" ]; then
    log_error "Directory '$PROJECT_NAME' already exists. Please remove it or rename it to proceed."
fi

# ------------------------------------------------------------------------------
# 4. Project Initialization
# ------------------------------------------------------------------------------
log_info "Initializing project structure..."
mkdir -p "$PROJECT_NAME"/{cmd/scraper,internal/{collector,domain,ingest,storage,dashboard},input,data/archive}
cd "$PROJECT_NAME"

# --- SECURITY STEP: Initialize Git & Ignore Secrets Immediately ---
log_info "Initializing version control..."
git init -q
cat <<EOF > .gitignore
# Secrets & Config
.env
config.yaml

# Data (Do not commit scraped data)
data/
*.json
*.csv
!input/*.csv

# OS & Editor
.DS_Store
.vscode/
.idea/
bin/
EOF
log_success "Git initialized and .gitignore created."

# --- Go Module Init ---
log_info "Initializing Go module..."
go mod init "$MODULE_NAME"

# ------------------------------------------------------------------------------
# 5. Source Code Generation
# ------------------------------------------------------------------------------
log_info "Generating source code..."

# --- DOMAIN LAYER ---
create_file "internal/domain/models.go" "$(cat <<EOF
package domain

import "context"

// Target represents a scraping task
type Target struct {
	Subreddit string
	MinScore  int
}

// Post is the clean data structure for storage
type Post struct {
	ID           string   \`json:"id"\`
	Title        string   \`json:"title"\`
	Subreddit    string   \`json:"subreddit"\`
	Author       string   \`json:"author"\`
	URL          string   \`json:"url"\`
	Score        int      \`json:"score"\`
	CommentCount int      \`json:"comment_count"\`
	CreatedUTC   float64  \`json:"created_utc"\`
	KeywordsHit  []string \`json:"keywords_hit,omitempty"\`
}

// Collector defines the interface for data fetching
type Collector interface {
	FetchNewPosts(ctx context.Context, subreddit string, limit int) ([]Post, error)
}
EOF
)"

# --- INGEST LAYER ---
create_file "internal/ingest/reader.go" "$(cat <<EOF
package ingest

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/qepting91/reddit-scraper/internal/domain"
)

// Regex for valid subreddit names
var subNameRegex = regexp.MustCompile(\`^[A-Za-z0-9_]{3,21}$\`)

func LoadTargets(path string) ([]domain.Target, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Wrap in BOM stripper
	r := csv.NewReader(stripBOM(f))
	
	var targets []domain.Target
	line := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		line++
		if line == 1 { continue } // Skip header

		// Validation (Fail-Soft)
		sub := strings.TrimSpace(record[0])
		if !subNameRegex.MatchString(sub) {
			continue 
		}

		score, _ := strconv.Atoi(strings.TrimSpace(record[1]))

		targets = append(targets, domain.Target{
			Subreddit: sub,
			MinScore:  score,
		})
	}
	return targets, nil
}

func LoadKeywords(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	r := csv.NewReader(stripBOM(f))
	var kws []string
	line := 0
	for {
		rec, err := r.Read()
		if err == io.EOF { break }
		if line > 0 && len(rec) > 0 {
			kws = append(kws, strings.ToLower(strings.TrimSpace(rec[0])))
		}
		line++
	}
	return kws, nil
}

func stripBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	rdr, _, err := br.ReadRune()
	if err != nil { return br }
	if rdr != '\uFEFF' { br.UnreadRune() }
	return br
}
EOF
)"

# --- COLLECTOR LAYER ---
create_file "internal/collector/client.go" "$(cat <<EOF
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

func NewClient(id, secret, user, pass string) (*RedditClient, error) {
	creds := reddit.Credentials{ID: id, Secret: secret, Username: user, Password: pass}
	
	// Using loganintech fork
	client, err := reddit.NewClient(creds, reddit.WithUserAgent("go:scraper:v1.0 (by /u/qepting91)"))
	if err != nil { return nil, err }

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
EOF
)"

# --- STORAGE LAYER ---
create_file "internal/storage/writer.go" "$(cat <<EOF
package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/qepting91/reddit-scraper/internal/domain"
)

// WriterService implements the Monitor Pattern for thread safety
type WriterService struct {
	FilePath string
}

func (w *WriterService) Start(wg *sync.WaitGroup, input <-chan domain.Post) {
	defer wg.Done()

	f, err := os.OpenFile(w.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)

	for post := range input {
		// Write as NDJSON
		enc.Encode(post)
	}
}
EOF
)"

# --- DASHBOARD LAYER ---
create_file "internal/dashboard/server.go" "$(cat <<EOF
package dashboard

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/qepting91/reddit-scraper/internal/domain"
)

func StartServer(dataFile string, port string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts := loadData(dataFile)
		
		// 1. Topic Dominance
		pie := charts.NewPie()
		pie.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Subreddit Dominance"}),
			charts.WithThemeOpts(opts.Theme{Theme: types.ThemeWesteros}),
		)
		
		subCounts := make(map[string]int)
		for _, p := range posts { subCounts[p.Subreddit]++ }
		
		var pieItems []opts.PieData
		for k, v := range subCounts {
			pieItems = append(pieItems, opts.PieData{Name: k, Value: v})
		}
		pie.AddSeries("Posts", pieItems)

		// 2. Keyword Velocity
		bar := charts.NewBar()
		bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{Title: "Keyword Velocity"}))
		
		kwCounts := make(map[string]int)
		for _, p := range posts {
			for _, k := range p.KeywordsHit {
				kwCounts[k]++
			}
		}
		
		var barX []string
		var barY []opts.BarData
		for k, v := range kwCounts {
			barX = append(barX, k)
			barY = append(barY, opts.BarData{Value: v})
		}
		bar.SetXAxis(barX).AddSeries("Mentions", barY)

		pie.Render(w)
		bar.Render(w)
	})

	return http.ListenAndServe(":"+port, nil)
}

func loadData(path string) []domain.Post {
	f, _ := os.Open(path)
	defer f.Close()
	var posts []domain.Post
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var p domain.Post
		if err := json.Unmarshal(scanner.Bytes(), &p); err == nil {
			posts = append(posts, p)
		}
	}
	return posts
}
EOF
)"

# --- MAIN ---
create_file "cmd/scraper/main.go" "$(cat <<EOF
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/qepting91/reddit-scraper/internal/collector"
	"github.com/qepting91/reddit-scraper/internal/dashboard"
	"github.com/qepting91/reddit-scraper/internal/domain"
	"github.com/qepting91/reddit-scraper/internal/ingest"
	"github.com/qepting91/reddit-scraper/internal/storage"
)

func main() {
	// 1. Setup
	godotenv.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Run Dashboard
	go func() {
		logger.Info("Starting Dashboard on :8080")
		if err := dashboard.StartServer("data/current.json", "8080"); err != nil {
			logger.Error("Dashboard failed", "err", err)
		}
	}()

	// 3. Load Inputs
	targets, _ := ingest.LoadTargets("input/subreddits.csv")
	keywords, _ := ingest.LoadKeywords("input/keywords.csv")

	// 4. Initialize Client
	client, err := collector.NewClient(
		os.Getenv("REDDIT_CLIENT_ID"),
		os.Getenv("REDDIT_CLIENT_SECRET"),
		os.Getenv("REDDIT_USERNAME"),
		os.Getenv("REDDIT_PASSWORD"),
	)
	if err != nil {
		logger.Error("Failed to auth with Reddit", "error", err)
		os.Exit(1)
	}

	// 5. Concurrency Setup (Fan-Out/Fan-In Pattern)
	jobQueue := make(chan domain.Target, len(targets))
	resultQueue := make(chan domain.Post, 100)
	var workerWg sync.WaitGroup
	var writerWg sync.WaitGroup

	// Start Writer (Monitor)
	writer := &storage.WriterService{FilePath: "data/current.json"}
	writerWg.Add(1)
	go writer.Start(&writerWg, resultQueue)

	// Start Workers
	ctx, cancel := context.WithCancel(context.Background())
	numWorkers := 4 
	
	for i := 0; i < numWorkers; i++ {
		workerWg.Add(1)
		go func(id int) {
			defer workerWg.Done()
			for t := range jobQueue {
				select {
				case <-ctx.Done():
					return
				default:
					// Fetch
					posts, err := client.FetchNewPosts(ctx, t.Subreddit, 25)
					if err != nil {
						logger.Error("Scrape failed", "sub", t.Subreddit, "err", err)
						continue
					}
					
					// Process & Filter
					for _, p := range posts {
						for _, k := range keywords {
							if strings.Contains(strings.ToLower(p.Title), k) {
								p.KeywordsHit = append(p.KeywordsHit, k)
							}
						}
						if p.Score >= t.MinScore || len(p.KeywordsHit) > 0 {
							resultQueue <- p
						}
					}
				}
			}
		}(i)
	}

	// 6. Enqueue Jobs
	logger.Info("Starting scrape cycle", "targets", len(targets))
	for _, t := range targets {
		jobQueue <- t
	}
	close(jobQueue)

	// 7. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	// Wait for completion
	workerWg.Wait()
	close(resultQueue) 
	writerWg.Wait()
	logger.Info("Scrape complete. Data saved.")
	
	// Keep alive for dashboard
	select{} 
}
EOF
)"

# ------------------------------------------------------------------------------
# 6. Dependency Management
# ------------------------------------------------------------------------------
log_info "Fetching dependencies..."
go get github.com/loganintech/go-reddit/v2@latest
go get github.com/robfig/cron/v3@latest
go get github.com/go-echarts/go-echarts/v2@latest
go get golang.org/x/time/rate@latest
go get github.com/joho/godotenv@latest

log_info "Cleaning up module..."
go mod tidy

# ------------------------------------------------------------------------------
# 7. Configuration & Test Data
# ------------------------------------------------------------------------------
log_info "Generating test data..."

create_file "input/subreddits.csv" "$(cat <<EOF
subreddit,min_score
golang,10
rust,15
cybersecurity,25
netsec,5
sysadmin,50
devops,10
EOF
)"

create_file "input/keywords.csv" "$(cat <<EOF
keyword,category
threat intelligence,security
zero day,security
concurrency,programming
memory leak,programming
rate limit,architecture
docker,devops
EOF
)"

# --- SECURITY STEP: Set Permissions on .env ---
log_info "Generating secure .env file..."
cat <<EOF > .env
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
REDDIT_USERNAME=your_username
REDDIT_PASSWORD=your_password
LOG_LEVEL=info
PORT=8080
EOF
chmod 600 .env
log_success ".env created with restricted permissions (600)."

# ------------------------------------------------------------------------------
# 8. Final Instructions
# ------------------------------------------------------------------------------
echo ""
log_success "Setup Complete!"
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. cd $PROJECT_NAME"
echo "  2. Edit .env with your actual Reddit credentials."
echo "  3. Run the scraper: go run cmd/scraper/main.go"
echo ""log_info "Happy Scraping!"