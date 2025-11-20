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
