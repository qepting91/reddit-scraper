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

	// FIX: Load Port from Env (Default to 8080 if missing)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Run Dashboard
	go func() {
		logger.Info("Starting Dashboard", "port", port)
		if err := dashboard.StartServer("data/current.json", port); err != nil {
			logger.Error("Dashboard failed", "err", err)
		}
	}()

	// 3. Load Inputs
	targets, _ := ingest.LoadTargets("input/subreddits.csv")
	keywords, _ := ingest.LoadKeywords("input/keywords.csv")

	// 4. Initialize Client (Using Factory)
	client, err := collector.NewCollector()
	if err != nil {
		logger.Error("Failed to initialize collector", "error", err)
		os.Exit(1)
	}
	logger.Info("Collector initialized", "mode", os.Getenv("COLLECTOR_MODE"))

	// 5. Concurrency Setup
	jobQueue := make(chan domain.Target, len(targets))
	resultQueue := make(chan domain.Post, 100)
	var workerWg sync.WaitGroup
	var writerWg sync.WaitGroup

	writer := &storage.WriterService{FilePath: "data/current.json"}
	writerWg.Add(1)
	go writer.Start(&writerWg, resultQueue)

	// Start Workers
	ctx, cancel := context.WithCancel(context.Background())

	// Adjust workers based on mode to prevent rate limiting
	numWorkers := 4
	if os.Getenv("COLLECTOR_MODE") == "public" {
		numWorkers = 2 // Go slower for public JSON
	}

	for i := 0; i < numWorkers; i++ {
		workerWg.Add(1)
		go func(id int) {
			defer workerWg.Done()
			for t := range jobQueue {
				select {
				case <-ctx.Done():
					return
				default:
					posts, err := client.FetchNewPosts(ctx, t.Subreddit, 25)
					if err != nil {
						logger.Error("Scrape failed", "sub", t.Subreddit, "err", err)
						continue
					}
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

	workerWg.Wait()
	close(resultQueue)
	writerWg.Wait()
	logger.Info("Scrape complete. Data saved.")

	// Keep alive for dashboard
	select {}
}
