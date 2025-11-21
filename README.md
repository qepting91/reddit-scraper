# 🛡️ Reddit Intelligence Monitor

A high-performance, concurrent intelligence gathering tool written in Go. This application monitors specific subreddits for mentions of defensive tools, threat actors, or specific keywords, visualizing the data in a real-time "Analyst Report" dashboard.

## 🚀 Overview

This tool is designed for security researchers and CTI analysts to track trends across technical communities. It supports multiple operation modes to suit different access levels:

* **Public Mode:** Scrapes Reddit JSON endpoints (No API keys required).
* **API Mode:** Uses the official Reddit API (Authenticated).
* **Mock Mode:** Generates synthetic data for testing.

## 🛠️ Installation & Setup

For detailed installation steps, configuration guides, and the automated setup script, please consult the **Installation Instructions** folder.

👉 **[View Installation Instructions](Installation_Instructions/readme.md)**

### Quick Start Script
If you are setting this up for the first time, you can find the automated scaffolding script here:

`Installation_Instructions/setup_project.sh`

## 📊 Features

* **Live Dashboard:** Visualizes tool popularity and subreddit activity.
* **Rate Limiting:** Built-in throttling to respect Reddit's API terms.
* **Exportable Data:** Saves all intelligence data to local JSON for further analysis.

## 📂 Repository Structure

* `cmd/`: Application entry point.
* `internal/`: Core application logic (Collector, Dashboard, Storage).
* `input/`: CSV configuration files for targets and keywords.
* `Installation_Instructions/`: Setup scripts and detailed documentation.