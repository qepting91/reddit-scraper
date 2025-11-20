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
var subNameRegex = regexp.MustCompile(`^[A-Za-z0-9_]{3,21}$`)

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
