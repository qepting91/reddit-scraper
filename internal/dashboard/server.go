package dashboard

import (
	"bufio"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/qepting91/reddit-scraper/internal/domain"
)

// DashboardView holds all data required by the HTML template
type DashboardView struct {
	PieSnippet  template.HTML
	BarSnippet  template.HTML
	HeatSnippet template.HTML
	Posts       []domain.Post
}

// boolPtr is a helper to satisfy go-echarts' requirement for *bool fields
func boolPtr(b bool) *bool {
	return &b
}

func StartServer(dataFile string, port string) error {
	// 1. Define the comprehensive HTML Template
	tpl := template.Must(template.New("dashboard").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Threat Intel Dashboard</title>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/echarts.min.js"></script>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/themes/westeros.js"></script>
    <style>
        body { background-color: #1a1a1a; color: #e0e0e0; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; margin: 0; padding: 20px; }
        h1, h2 { color: #fff; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .container { max-width: 1600px; margin: 0 auto; }
        .charts-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; }
        .full-width { grid-column: 1 / -1; }
        
        /* Table Styling */
        .data-table { width: 100%; border-collapse: collapse; background: #252525; box-shadow: 0 4px 6px rgba(0,0,0,0.3); border-radius: 8px; overflow: hidden; }
        .data-table th { background: #333; text-align: left; padding: 12px; color: #4fc3f7; }
        .data-table td { padding: 12px; border-bottom: 1px solid #333; }
        .data-table tr:hover { background: #2a2a2a; }
        a { color: #4fc3f7; text-decoration: none; }
        a:hover { text-decoration: underline; color: #81d4fa; }
        .tag { background: #383838; padding: 2px 8px; border-radius: 4px; font-size: 0.85em; margin-right: 4px; border: 1px solid #444; }
        .score { font-weight: bold; color: #ffb74d; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🛡️ Reddit Threat Intelligence</h1>
        
        <div class="charts-grid">
            <div class="chart-box">{{.PieSnippet}}</div>
            <div class="chart-box">{{.BarSnippet}}</div>
            <div class="chart-box full-width">{{.HeatSnippet}}</div>
        </div>

        <h2>🔍 Live Feed & Links</h2>
        <table class="data-table">
            <thead>
                <tr>
                    <th>Score</th>
                    <th>Subreddit</th>
                    <th>Title (Click to View)</th>
                    <th>Keywords Hit</th>
                    <th>Author</th>
                </tr>
            </thead>
            <tbody>
                {{range .Posts}}
                <tr>
                    <td class="score">⬆ {{.Score}}</td>
                    <td><a href="https://reddit.com/{{.Subreddit}}" target="_blank">{{.Subreddit}}</a></td>
                    <td><a href="{{.URL}}" target="_blank">{{.Title}}</a></td>
                    <td>
                        {{range .KeywordsHit}}<span class="tag">{{.}}</span>{{end}}
                    </td>
                    <td>{{.Author}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts := loadData(dataFile)

		// Sort posts by score (descending) for the table
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].Score > posts[j].Score
		})

		// 1. Pie Chart (Subreddit Volume)
		pie := charts.NewPie()
		pie.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Post Volume by Subreddit"}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
			charts.WithLegendOpts(opts.Legend{Show: boolPtr(false)}),
		)
		subCounts := make(map[string]int)
		for _, p := range posts {
			subCounts[p.Subreddit]++
		}
		var pieItems []opts.PieData
		for k, v := range subCounts {
			pieItems = append(pieItems, opts.PieData{Name: k, Value: v})
		}
		pie.AddSeries("Posts", pieItems)

		// 2. Bar Chart (Keyword Velocity)
		bar := charts.NewBar()
		bar.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Top Detected Threats"}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		)
		kwCounts := make(map[string]int)
		for _, p := range posts {
			for _, k := range p.KeywordsHit {
				kwCounts[k]++
			}
		}
		var barX []string
		for k := range kwCounts {
			barX = append(barX, k)
		}
		sort.Strings(barX)

		var barY []opts.BarData
		for _, k := range barX {
			barY = append(barY, opts.BarData{Value: kwCounts[k]})
		}
		bar.SetXAxis(barX).AddSeries("Mentions", barY)

		// 3. Heatmap (Keywords vs Subreddits)
		heatmap := charts.NewHeatMap()
		heatmap.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Threat Heatmap (Keywords vs Subreddits)"}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
			charts.WithTooltipOpts(opts.Tooltip{Show: boolPtr(true)}),
			charts.WithVisualMapOpts(opts.VisualMap{
				Calculable: boolPtr(true),
				Min:        0,
				Max:        5,
				InRange:    &opts.VisualMapInRange{Color: []string{"#37474f", "#0288d1", "#e65100"}},
			}),
		)

		// Prepare Heatmap Data
		uniqueSubsMap := make(map[string]bool)
		uniqueKwsMap := make(map[string]bool)
		grid := make(map[string]map[string]int)

		for _, p := range posts {
			uniqueSubsMap[p.Subreddit] = true
			if _, exists := grid[p.Subreddit]; !exists {
				grid[p.Subreddit] = make(map[string]int)
			}
			for _, k := range p.KeywordsHit {
				uniqueKwsMap[k] = true
				grid[p.Subreddit][k]++
			}
		}

		var xSubs []string
		for k := range uniqueSubsMap {
			xSubs = append(xSubs, k)
		}
		sort.Strings(xSubs)

		var yKws []string
		for k := range uniqueKwsMap {
			yKws = append(yKws, k)
		}
		sort.Strings(yKws)

		var hmData []opts.HeatMapData
		for i, sub := range xSubs {
			for j, kw := range yKws {
				val := grid[sub][kw]
				hmData = append(hmData, opts.HeatMapData{Value: []interface{}{i, j, val}})
			}
		}

		// FIXED: Use SetXAxis for X, and WithYAxisOpts for Y (since SetYAxis is missing in v2)
		heatmap.SetXAxis(xSubs)
		heatmap.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				Data:      yKws,
				Type:      "category", // Essential for text labels on Y-axis
				SplitArea: &opts.SplitArea{Show: boolPtr(true)},
			}),
		)
		heatmap.AddSeries("Overlap", hmData)

		// Render Charts to HTML Snippets
		view := DashboardView{
			PieSnippet:  renderSnippet(pie),
			BarSnippet:  renderSnippet(bar),
			HeatSnippet: renderSnippet(heatmap),
			Posts:       posts,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := tpl.Execute(w, view); err != nil {
			log.Printf("Template error: %v", err)
		}
	})

	return http.ListenAndServe(":"+port, nil)
}

// Interface matching the library's RenderSnippet signature
type snippetRenderer interface {
	RenderSnippet() render.ChartSnippet
}

// Helper to render just the chart DIV and Script
func renderSnippet(c snippetRenderer) template.HTML {
	s := c.RenderSnippet()
	return template.HTML(s.Element + "\n" + s.Script)
}

func loadData(path string) []domain.Post {
	f, err := os.Open(path)
	if err != nil {
		return []domain.Post{}
	}
	defer f.Close()

	var posts []domain.Post
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var p domain.Post
		if err := json.Unmarshal(line, &p); err == nil {
			posts = append(posts, p)
		}
	}
	return posts
}
