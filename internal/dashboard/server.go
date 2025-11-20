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

// DashboardView holds data for the HTML template, including new KPIs
type DashboardView struct {
	PieSnippet   template.HTML
	BarSnippet   template.HTML
	HeatSnippet  template.HTML
	Posts        []domain.Post
	TotalPosts   int
	TopThreat    string
	HottestSub   string
	HighestScore int
}

// boolPtr helper
func boolPtr(b bool) *bool { return &b }

func StartServer(dataFile string, port string) error {
	// 1. Define the HTML Template with KPI Cards and JS Filtering
	tpl := template.Must(template.New("dashboard").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Threat Intel Dashboard</title>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/echarts.min.js"></script>
    <script src="https://go-echarts.github.io/go-echarts-assets/assets/themes/westeros.js"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;800&display=swap" rel="stylesheet">
    <style>
        body { background-color: #0f172a; color: #cbd5e1; font-family: 'Inter', sans-serif; margin: 0; padding: 20px; }
        .container { max-width: 1600px; margin: 0 auto; }
        
        /* Header & KPIs */
        header { margin-bottom: 30px; border-bottom: 1px solid #334155; padding-bottom: 20px; }
        h1 { color: #f8fafc; margin: 0; font-size: 1.8rem; }
        .kpi-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-top: 20px; }
        .kpi-card { background: #1e293b; padding: 20px; border-radius: 12px; border: 1px solid #334155; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); }
        .kpi-label { color: #94a3b8; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
        .kpi-value { color: #38bdf8; font-size: 2rem; font-weight: 800; margin-top: 5px; }
        .kpi-value.danger { color: #f87171; }

        /* Charts */
        .charts-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 30px; }
        .chart-box { background: #1e293b; border-radius: 12px; padding: 15px; border: 1px solid #334155; overflow: hidden; }
        .full-width { grid-column: 1 / -1; }

        /* Table */
        h2 { color: #f8fafc; font-size: 1.4rem; margin-bottom: 15px; display: flex; align-items: center; gap: 10px; }
        .table-container { overflow-x: auto; border-radius: 12px; border: 1px solid #334155; }
        table { width: 100%; border-collapse: collapse; background: #1e293b; }
        th { background: #0f172a; text-align: left; padding: 16px; color: #94a3b8; font-weight: 600; border-bottom: 1px solid #334155; }
        td { padding: 16px; border-bottom: 1px solid #334155; color: #e2e8f0; }
        tr:hover { background: #334155; transition: background 0.2s; }
        
        /* Utilities */
        a { color: #38bdf8; text-decoration: none; font-weight: 500; }
        a:hover { text-decoration: underline; color: #7dd3fc; }
        .tag { background: #334155; color: #cbd5e1; padding: 4px 10px; border-radius: 999px; font-size: 0.75rem; margin-right: 5px; display: inline-block; border: 1px solid #475569; }
        .score-high { color: #f87171; font-weight: bold; } 
        .score-med { color: #fbbf24; font-weight: bold; } 
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🛡️ Threat Intel Command Center</h1>
            <div class="kpi-grid">
                <div class="kpi-card">
                    <div class="kpi-label">Total Intel Reports</div>
                    <div class="kpi-value">{{.TotalPosts}}</div>
                </div>
                <div class="kpi-card">
                    <div class="kpi-label">Top Detected Threat</div>
                    <div class="kpi-value danger">{{.TopThreat}}</div>
                </div>
                <div class="kpi-card">
                    <div class="kpi-label">Most Active Sector</div>
                    <div class="kpi-value">{{.HottestSub}}</div>
                </div>
                <div class="kpi-card">
                    <div class="kpi-label">Highest Impact Score</div>
                    <div class="kpi-value">{{.HighestScore}}</div>
                </div>
            </div>
        </header>
        
        <div class="charts-grid">
            <div class="chart-box">{{.PieSnippet}}</div>
            <div class="chart-box" id="bar-container">{{.BarSnippet}}</div>
            <div class="chart-box full-width">{{.HeatSnippet}}</div>
        </div>

        <h2>
            🔍 Live Intelligence Feed 
            <span style="font-size: 0.8rem; background: #334155; padding: 4px 8px; border-radius: 4px; color: #94a3b8; margin-left: auto;">
                Tip: Click a bar in the "Top Threats" chart to filter this table
            </span>
        </h2>
        <div class="table-container">
            <table id="feedTable">
                <thead>
                    <tr>
                        <th>Impact</th>
                        <th>Source</th>
                        <th>Intelligence Summary</th>
                        <th>Tags</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Posts}}
                    <tr data-keywords="{{range .KeywordsHit}}{{.}} {{end}}">
                        <td class="{{if ge .Score 100}}score-high{{else if ge .Score 50}}score-med{{end}}">
                            {{.Score}}
                        </td>
                        <td><a href="https://reddit.com/{{.Subreddit}}" target="_blank">{{.Subreddit}}</a></td>
                        <td><a href="{{.URL}}" target="_blank">{{.Title}}</a></td>
                        <td>
                            {{range .KeywordsHit}}<span class="tag">{{.}}</span>{{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>

    <script>
        // 1. Filter Function
        function filterTable(keyword) {
            const rows = document.querySelectorAll("#feedTable tbody tr");
            keyword = keyword.toLowerCase();
            rows.forEach(row => {
                const tags = row.getAttribute("data-keywords").toLowerCase();
                if (tags.includes(keyword) || keyword === "") {
                    row.style.display = "";
                } else {
                    row.style.display = "none";
                }
            });
        }

        // 2. Event Listener Attachment (Manual Method)
        window.addEventListener('load', function() {
            // We use a timeout to ensure the echarts library has fully initialized the DOM
            setTimeout(function() {
                // Retrieve the chart instance by the fixed ID we set in Go
                const chartDom = document.getElementById('bar-chart-id');
                if (chartDom) {
                    const myChart = echarts.getInstanceByDom(chartDom);
                    if (myChart) {
                        myChart.on('click', function(params) {
                            filterTable(params.name);
                        });
                        console.log("✅ Interactivity enabled for Bar Chart");
                    }
                }
            }, 1000);
        });
    </script>
</body>
</html>
`))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts := loadData(dataFile)

		// --- 1. Calculate KPIs ---
		totalPosts := len(posts)
		highestScore := 0
		subCounts := make(map[string]int)
		kwCounts := make(map[string]int)

		for _, p := range posts {
			if p.Score > highestScore {
				highestScore = p.Score
			}
			subCounts[p.Subreddit]++
			for _, k := range p.KeywordsHit {
				kwCounts[k]++
			}
		}

		topThreat := "None"
		maxKwCount := 0
		for k, v := range kwCounts {
			if v > maxKwCount {
				maxKwCount = v
				topThreat = k
			}
		}

		hottestSub := "None"
		maxSubCount := 0
		for k, v := range subCounts {
			if v > maxSubCount {
				maxSubCount = v
				hottestSub = k
			}
		}

		// --- 2. Generate Charts ---

		// PIE CHART
		pie := charts.NewPie()
		pie.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Volume by Source"}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros, Height: "350px"}),
			charts.WithLegendOpts(opts.Legend{Show: boolPtr(false)}),
		)
		var pieItems []opts.PieData
		for k, v := range subCounts {
			pieItems = append(pieItems, opts.PieData{Name: k, Value: v})
		}
		pie.AddSeries("Posts", pieItems)

		// BAR CHART
		bar := charts.NewBar()
		bar.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Top Detected Threats"}),
			// FIXED: Set a fixed ChartID so we can grab it with JS later
			charts.WithInitializationOpts(opts.Initialization{
				Theme:   types.ThemeWesteros,
				Height:  "350px",
				ChartID: "bar-chart-id",
			}),
		)
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

		// HEATMAP
		heatmap := charts.NewHeatMap()
		heatmap.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{Title: "Threat Heatmap"}),
			charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros, Height: "400px"}),
			charts.WithVisualMapOpts(opts.VisualMap{
				Calculable: boolPtr(true),
				Min:        0, Max: 5,
				InRange: &opts.VisualMapInRange{Color: []string{"#1e293b", "#0ea5e9", "#ef4444"}},
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
		heatmap.SetXAxis(xSubs)
		heatmap.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{Data: yKws, Type: "category", SplitArea: &opts.SplitArea{Show: boolPtr(true)}}),
		)
		heatmap.AddSeries("Overlap", hmData)

		// --- 3. Render View ---
		sort.Slice(posts, func(i, j int) bool { return posts[i].Score > posts[j].Score })

		view := DashboardView{
			PieSnippet:   renderSnippet(pie),
			BarSnippet:   renderSnippet(bar),
			HeatSnippet:  renderSnippet(heatmap),
			Posts:        posts,
			TotalPosts:   totalPosts,
			TopThreat:    topThreat,
			HottestSub:   hottestSub,
			HighestScore: highestScore,
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
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var p domain.Post
		if err := json.Unmarshal(scanner.Bytes(), &p); err == nil {
			posts = append(posts, p)
		}
	}
	return posts
}
