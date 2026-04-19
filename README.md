# 🎵 Groupie Tracker

A web application built in Go that lets you explore music artists and bands using the Groupie Trackers API. Browse artists, view concert locations, dates, and tour schedules — all in a clean, modern UI.

## Features

- 🎤 Browse all artists and bands
- 🔍 Search bar with live suggestions (artist name, members, locations, dates)
- 📍 Concert locations and tour schedules
- 📅 Concert dates per artist
- 🔎 Filter by creation year and number of members
- 📄 Pagination
- 📱 Responsive design

## Project Structure
groupie-tracker/
├── main.go
├── go.mod
├── templates/
│   ├── index.html
│   ├── artist.html
│   └── error.html
└── static/
├── css/
│   └── style.css
└── images/
└── cover.jpg
## Requirements

- Go 1.21 or higher
- Internet connection (fetches data from Groupie Trackers API)

## Installation & Run

1. Clone the repository:
```bash
git clone https://github.com/yourname/groupie-tracker.git
cd groupie-tracker
```

2. Run the server:
```bash
go run main.go
```

3. Open your browser at: http://localhost:8080
## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page — all artists |
| GET | `/artist/{id}` | Artist detail page |
| GET | `/api/artist/{id}` | Artist data as JSON |
| GET | `/api/search?q=` | Search suggestions |

## Search

The search bar supports the following categories:

| Category | Example |
|----------|---------|
| `artist/band` | Queen, The Beatles |
| `member` | Freddie Mercury |
| `location` | london-uk |
| `first album` | 13/07/1973 |
| `creation date` | 1970 |

