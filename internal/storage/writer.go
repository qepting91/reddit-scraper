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
