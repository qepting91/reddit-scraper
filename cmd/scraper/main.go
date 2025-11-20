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
