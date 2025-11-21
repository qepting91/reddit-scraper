# 🛡️ Reddit Intelligence Monitor

A high-performance, concurrent intelligence gathering tool written in Go. This application monitors specific subreddits for mentions of defensive tools, threat actors, or specific keywords, visualizing the data in a real-time "Analyst Report" dashboard.

## 🚀 Key Features

* **Multi-Mode Collection:**
    * **Public Mode:** Scrapes Reddit JSON endpoints (No API keys required).
    * **API Mode:** Uses the official Reddit API (requires credentials).
    * **Mock Mode:** Generates synthetic data for UI testing and development.
* **Analysis Dashboard:** A clean, web-based report showing tool distribution, velocity, and top trends.
* **Concurrent Architecture:** Uses a Fan-Out/Fan-In worker pool pattern for efficient data processing.
* **Rate Limit Aware:** Automatically throttles requests to prevent IP bans.

## 📋 Prerequisites

* **Go:** Version 1.24 or higher.
* **Git**

## ⚙️ Configuration

The application is controlled via a `.env` file in the root directory.

### 1. Select Operation Mode
Change `COLLECTOR_MODE` to switch how data is gathered.

```text
# Options: 'public', 'api', 'mock'
COLLECTOR_MODE=public
````

### 2\. Configure Credentials

**For Public Mode (Recommended for beginners):**
You only need to set a User Agent. Use your actual Reddit username.

```text
REDDIT_USER_AGENT="desktop:intel-monitor:v1.0 (by /u/YourUsername)"
```

**For API Mode:**
Required only if you have a registered Reddit App.

```text
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_client_secret
REDDIT_USERNAME=your_username
REDDIT_PASSWORD=your_password
```

## 🏃 Usage

1.  **Install Dependencies:**

    ```text
    go mod tidy
    ```

2.  **Run the Monitor:**

    ```text
    go run cmd/scraper/main.go
    ```

3.  **View the Report:**
    Open your browser to `http://localhost:8080` (or the port defined in your .env).

## 📊 Input Configuration

You can customize what the scraper looks for by editing the CSV files in the `input/` directory.

  * **`input/subreddits.csv`**: The communities to scan.
    ```text
    subreddit,min_score
    netsec,10
    threatintel,5
    ```
  * **`input/keywords.csv`**: The tools or terms to track.
    ```text
    keyword,category
    Splunk,tool
    CrowdStrike,tool
    ```

## 📂 Project Structure

```text
reddit-scraper/
├── cmd/scraper/           # Application entry point
├── internal/
│   ├── collector/         # Logic for fetching data (Public/API/Mock)
│   ├── dashboard/         # Web server and HTML rendering
│   ├── domain/            # Data models
│   ├── ingest/            # CSV reading logic
│   └── storage/           # JSON file saving
├── input/                 # Configuration CSVs
└── data/                  # Scraped results (current.json)
```

## ⚠️ Disclaimer

This tool is for educational and research purposes. When using Public Mode, ensure you respect Reddit's rate limits (handled automatically by this tool) and Terms of Service.

```
