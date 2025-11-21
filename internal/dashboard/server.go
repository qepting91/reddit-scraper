package dashboard

import (
	"bufio"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/qepting91/reddit-scraper/internal/domain"
)

// DashboardView holds data for the HTML template
type DashboardView struct {
	StackedBarSnippet template.HTML
	Posts             []domain.Post
	TotalMentions     int
	TopTool           string
	TopSub            string
	HighestScore      int
}

func boolPtr(b bool) *bool { return &b }

func StartServer(dataFile string, port string) error {
	// Clean, high-contrast "Analyst Report" template
	tpl := template.Must(template.New("dashboard").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Tool Monitor Report</title>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/echarts.min.js"></script>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/themes/westeros.js"></script>
    <style>
        :root { --bg: #f3f4f6; --card: #ffffff; --text: #111827; --border: #e5e7eb; --blue: #2563eb; }
        body { background-color: var(--bg); color: var(--text); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 30px; }
        .container { max-width: 1400px; margin: 0 auto; }
        
        /* Header Section */
        .header { background: var(--card); padding: 20px 30px; border-radius: 8px; box-shadow: 0 1px 2px rgba(0,0,0,0.05); margin-bottom: 25px; display: flex; justify-content: space-between; align-items: center; }
        h1 { margin: 0; font-size: 1.5rem; font-weight: 700; color: #1f2937; }
        .subtitle { font-size: 0.875rem; color: #6b7280; margin-top: 4px; }

        /* KPI Cards */
        .stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-bottom: 25px; }
        .stat-card { background: var(--card); padding: 20px; border-radius: 8px; border: 1px solid var(--border); }
        .stat-label { font-size: 0.75rem; text-transform: uppercase; font-weight: 600; color: #6b7280; letter-spacing: 0.05em; }
        .stat-value { font-size: 1.75rem; font-weight: 800; color: #111827; margin-top: 8px; }
        .highlight { color: var(--blue); }

        /* Chart Section */
        .chart-section { background: var(--card); padding: 20px; border-radius: 8px; border: 1px solid var(--border); margin-bottom: 25px; }
        .chart-title { font-size: 1rem; font-weight: 600; margin-bottom: 15px; color: #374151; }
        
        /* Table Section */
        .table-section { background: var(--card); border-radius: 8px; border: 1px solid var(--border); overflow: hidden; }
        table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
        th { background: #f9fafb; text-align: left; padding: 12px 20px; border-bottom: 1px solid var(--border); color: #4b5563; font-weight: 600; }
        td { padding: 12px 20px; border-bottom: 1px solid var(--border); color: #374151; }
        tr:hover { background: #f9fafb; }
        
        /* Tags & Links */
        .tag { background: #eff6ff; color: #1d4ed8; padding: 2px 10px; border-radius: 999px; font-size: 0.75rem; font-weight: 500; border: 1px solid #dbeafe; margin-right: 5px; display: inline-block; }
        .score { font-family: monospace; font-weight: 700; color: #059669; background: #d1fae5; padding: 2px 6px; border-radius: 4px; }
        a { color: #2563eb; text-decoration: none; font-weight: 500; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div>
                <h1>Intelligence Monitor</h1>
                <div class="subtitle">Tracking tool mentions across technical subreddits</div>
            </div>
            <div style="text-align: right;">
                <div class="stat-label">Data Source</div>
                <div style="font-weight: 600;">Local JSON</div>
            </div>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Mentions</div>
                <div class="stat-value">{{.TotalMentions}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Most Discussed Tool</div>
                <div class="stat-value highlight">{{.TopTool}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Most Active Subreddit</div>
                <div class="stat-value">{{.TopSub}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Highest Post Upvotes</div>
                <div class="stat-value">{{.HighestScore}}</div>
            </div>
        </div>

        <div class="chart-section">
            <div class="chart-title">Tool Distribution by Subreddit (Who is talking about what?)</div>
            {{.StackedBarSnippet}}
        </div>

        <div class="table-section">
            <table>
                <thead>
                    <tr>
                        <th width="100">Upvotes</th>
                        <th width="150">Subreddit</th>
                        <th>Post Title</th>
                        <th>Tools Mentioned</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Posts}}
                    <tr>
                        <td><span class="score">⬆ {{.Score}}</span></td>
                        <td><a href="https://reddit.com/{{.Subreddit}}" target="_blank">r/{{.Subreddit}}</a></td>
                        <td><a href="{{.URL}}" target="_blank" style="color: #111827; font-weight: 400;">{{.Title}}</a></td>
                        <td>
                            {{range .KeywordsHit}}<span class="tag">{{.}}</span>{{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>
`))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts := loadData(dataFile)

		// --- 1. Aggregation ---
		subCounts := make(map[string]int)
		toolCounts := make(map[string]int)
		// matrix[Subreddit][Tool] = Count
		matrix := make(map[string]map[string]int)

		uniqueSubs := make(map[string]bool)
		uniqueTools := make(map[string]bool)
		highestScore := 0

		for _, p := range posts {
			if p.Score > highestScore {
				highestScore = p.Score
			}
			sub := p.Subreddit

			subCounts[sub]++
			uniqueSubs[sub] = true

			if _, ok := matrix[sub]; !ok {
				matrix[sub] = make(map[string]int)
			}

			for _, k := range p.KeywordsHit {
				toolCounts[k]++
				uniqueTools[k] = true
				matrix[sub][k]++
			}
		}

		// --- 2. KPI Calculation ---
		topTool := "N/A"
		maxT := 0
		for k, v := range toolCounts {
			if v > maxT {
				maxT = v
				topTool = k
			}
		}

		topSub := "N/A"
		maxS := 0
		for k, v := range subCounts {
			if v > maxS {
				maxS = v
				topSub = k
			}
		}

		// --- 3. Chart Preparation ---

		// Sort Subreddits (X-Axis) Alphabetically
		var xSubs []string
		for s := range uniqueSubs {
			xSubs = append(xSubs, s)
		}
		sort.Strings(xSubs)

		// Sort Tools (Series) Alphabetically
		var tools []string
		for t := range uniqueTools {
			tools = append(tools, t)
		}
		sort.Strings(tools)

		// Create Stacked Bar Chart
		bar := charts.NewBar()
		bar.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{
				Theme:  types.ThemeWesteros,
				Height: "500px", // Ensure height is set so it's not an empty box
			}),
			charts.WithTooltipOpts(opts.Tooltip{Show: boolPtr(true), Trigger: "axis", AxisPointer: &opts.AxisPointer{Type: "shadow"}}),
			charts.WithLegendOpts(opts.Legend{Show: boolPtr(true), Bottom: "0"}),
			charts.WithXAxisOpts(opts.XAxis{AxisLabel: &opts.AxisLabel{Rotate: 45}}),
			charts.WithGridOpts(opts.Grid{Bottom: "15%", ContainLabel: boolPtr(true)}), // Prevent labels from being cut off
		)

		bar.SetXAxis(xSubs)

		// Add a series for each Tool found
		for _, tool := range tools {
			var data []opts.BarData
			for _, sub := range xSubs {
				// Get count for this specific Sub/Tool combo
				val := matrix[sub][tool]
				data = append(data, opts.BarData{Value: val})
			}
			// Stack: "total" forces them to stack
			bar.AddSeries(tool, data).SetSeriesOptions(
				charts.WithBarChartOpts(opts.BarChart{Stack: "total"}),
			)
		}

		// --- 4. Render ---
		view := DashboardView{
			StackedBarSnippet: renderSnippet(bar),
			Posts:             posts,
			TotalMentions:     len(posts),
			TopTool:           topTool,
			TopSub:            topSub,
			HighestScore:      highestScore,
		}

		w.Header().Set("Content-Type", "text/html")
		tpl.Execute(w, view)
	})

	return http.ListenAndServe(":"+port, nil)
}

type snippetRenderer interface {
	RenderSnippet() render.ChartSnippet
}

func renderSnippet(c snippetRenderer) template.HTML {
	s := c.RenderSnippet()
	return template.HTML(s.Element + "\n" + s.Script)
}

func loadData(path string) []domain.Post {
	f, err := os.Open(path)
	if err != nil {
		// Fail gracefully if file doesn't exist yet
		return []domain.Post{}
	}
	defer f.Close()

	var posts []domain.Post
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var p domain.Post
		if err := json.Unmarshal(scanner.Bytes(), &p); err == nil {
			posts = append(posts, p)
		}
	}
	// Sort by Score Descending
	sort.Slice(posts, func(i, j int) bool { return posts[i].Score > posts[j].Score })
	return posts
}
