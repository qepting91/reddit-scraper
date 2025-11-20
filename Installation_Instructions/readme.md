Reddit Threat Intelligence Scraper

A high-performance, concurrent data acquisition daemon written in Go. This tool is designed for security researchers and threat intelligence analysts to monitor high-signal technical subreddits (e.g., r/threatintel, r/netsec) for Indicators of Compromise (IOCs), zero-day exploits, and emerging malware trends.

🚀 Features

Concurrent Architecture: Uses a Fan-Out/Fan-In worker pool pattern to process multiple subreddits simultaneously.

Rate Limit Compliance: Implements a Token Bucket algorithm to strictly adhere to Reddit's API limits (60 requests/min).

Clean Architecture: Separation of concerns between ingestion, domain logic, and storage.

Fail-Soft Ingestion: Robust CSV parsing that handles BOM markers and formatting errors gracefully.

Visualization: Built-in HTML dashboard using go-echarts to track topic velocity and keyword dominance.

📋 Prerequisites

Go: Version 1.23 or higher.

Git: For version control.

Reddit Account: Required for API access.

⚙️ Installation & Setup

You can set up the project manually or use the included helper script.

1. Get the Code

git clone [https://github.com/qepting91/reddit-scraper.git](https://github.com/qepting91/reddit-scraper.git)
cd reddit-scraper


2. Generate Reddit API Credentials

To use this scraper, you must first register a "Script" application with Reddit to get your keys.

Log in to your Reddit account.

Navigate to the App Preferences page.

Click the button labeled "are you a developer? create an app..." (or "create another app").

Fill out the form exactly as follows:

Name: ThreatIntelMonitor (or your preferred name)

App Type: Select ● script (This is mandatory).

Redirect URI: http://localhost:8080

Description: (Optional) "Research tool for monitoring security feeds."

Click Create app.

Note your keys:

Client ID: The random string listed directly under the app name.

Client Secret: The long string labeled secret.

3. Submit API Access Request (Mandatory)

Reddit now requires a formal access request for many developer accounts.

Go to the Reddit Request Support Page.

In the "What do you need assistance with?" dropdown, select: API Access Request.

Fill out the form with the following suggested details to ensure approval for a research tool:

Form Field

Suggested Response

Role

I'm a developer

Inquiry

I’m a developer and want to build a Reddit App that does not work in the Devvit ecosystem.

Benefit/Purpose

"This application is a read-only internal research tool designed to monitor technical trends and threat intelligence feeds. It allows me to stay informed about emerging APT groups and zero-day exploits. It does not post content or spam users."

Detailed Description

"The app is a Go (Golang) daemon that strictly follows the 60 req/min rate limit. It polls specific subreddits for keywords (e.g. 'Ransomware') and saves metadata to local JSON storage for analysis. It performs READ-ONLY operations."

Missing from Devvit?

"My app requires external persistence (local filesystem/JSON) for archival and runs on a custom runtime (compiled binary on private infrastructure) which is not supported by Devvit's serverless environment."

Subreddits

r/threatintel, r/malware, r/netsec, r/cybersecurity, r/blueteamsec

4. Configure Environment Variables

Create a .env file in the project root to store your credentials securely.

cp .env.example .env  # If you have an example file, otherwise create one
nano .env


Paste the following configuration:

# Reddit API Credentials
REDDIT_CLIENT_ID=your_client_id_here
REDDIT_CLIENT_SECRET=your_client_secret_here
REDDIT_USERNAME=your_reddit_username
REDDIT_PASSWORD=your_reddit_password

# Application Settings
LOG_LEVEL=info
PORT=8080


⚠️ Security Warning: Never commit your .env file to GitHub. It contains your password.

5. Install Dependencies

Download the required Go modules:

go mod tidy


6. Configure Targets

Edit the input CSV files to customize your monitoring:

input/subreddits.csv: List of communities to scrape and minimum score thresholds.

input/keywords.csv: List of terms to flag (e.g., "Ransomware", "CVE-2024").

🏃 Usage

Run the Scraper

Execute the application from the command line:

go run cmd/scraper/main.go


Automated Setup (Optional)

If you need to scaffold the project structure from scratch or reset your environment, you can use the helper script located in the installation folder:

chmod +x installation/setup_project.sh
./installation/setup_project.sh


📊 Dashboard

Once the scraper is running, it will launch a local web server.
Open your browser to: http://localhost:8080

Subreddit Dominance: Pie chart showing volume of relevant posts per community.

Keyword Velocity: Bar chart showing frequency of your tracked threat terms.

📂 Project Structure

reddit-scraper/
├── cmd/
│   └── scraper/       # Main entry point
├── internal/
│   ├── collector/     # Reddit API Adapter
│   ├── dashboard/     # Visualization Server
│   ├── domain/        # Core Logic & Models
│   ├── ingest/        # CSV Processing
│   └── storage/       # JSON Persistence
├── input/             # Configuration CSVs
├── data/              # Output JSON files
└── installation/      # Setup scripts
