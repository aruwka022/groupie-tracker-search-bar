package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

const baseURL = "https://groupietrackers.herokuapp.com/api"

type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type LocationData struct {
	Locations []string `json:"locations"`
}

type ConcertDates struct {
	Dates []string `json:"dates"`
}

type RelationData struct {
	DatesLocations map[string][]string `json:"datesLocations"`
}

type Relation struct {
	Location string
	Dates    []string
}

type ExtendedArtist struct {
	ID           int
	Image        string
	Name         string
	Members      []string
	CreationDate int
	FirstAlbum   string
	ConcertDates []string
	Locations    []string
	Relations    []Relation
}

type SearchSuggestion struct {
	Label    string `json:"label"`
	Type     string `json:"type"`
	ArtistID int    `json:"artistId"`
}

type AllLocationsResponse struct {
	Index []struct {
		ID        int      `json:"id"`
		Locations []string `json:"locations"`
	} `json:"index"`
}

func main() {
	funcMap := template.FuncMap{
		"getUniqueYears": func(artists []Artist) []int {
			yearMap := make(map[int]bool)
			for _, artist := range artists {
				yearMap[artist.CreationDate] = true
			}
			years := make([]int, 0, len(yearMap))
			for year := range yearMap {
				years = append(years, year)
			}
			sort.Ints(years)
			return years
		},
	}

	indexTemplate := template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html"))
	artistTemplate := template.Must(template.New("artist.html").ParseFiles("templates/artist.html"))
	errorTemplate := template.Must(template.ParseFiles("templates/error.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleArtists(w, r, indexTemplate, errorTemplate)
		} else {
			handleNotFoundRoute(w, r, errorTemplate)
		}
	})

	http.HandleFunc("/artist/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handleError(w, errorTemplate, 400, "Bad Request")
			return
		}
		handleArtistDetail(w, r, artistTemplate, errorTemplate)
	})
	http.HandleFunc("/artist", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handleError(w, errorTemplate, 400, "Bad Request")
			return
		}
		handleArtistDetail(w, r, artistTemplate, errorTemplate)
	})

	http.HandleFunc("/api/artist/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleArtistJSON(w, r)
	})

	http.HandleFunc("/api/search", handleSearch)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server running at: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleArtists(w http.ResponseWriter, r *http.Request, tmpl *template.Template, errorTmpl *template.Template) {
	if r.Method != http.MethodGet {
		handleError(w, errorTmpl, 400, "Bad Request")
		return
	}

	artists, err := fetchAllArtists()
	if err != nil {
		log.Printf("Error fetching artists: %v", err)
		handleError(w, errorTmpl, 500, "Error fetching data from API")
		return
	}

	queenImage := ""
	for _, a := range artists {
		if strings.ToLower(strings.TrimSpace(a.Name)) == "queen" && a.Image != "" {
			queenImage = a.Image
			break
		}
	}
	if queenImage != "" {
		for i := range artists {
			if strings.TrimSpace(artists[i].Image) == "" {
				artists[i].Image = queenImage
			}
		}
	}

	data := struct {
		Title   string
		Artists []Artist
	}{
		Title:   "Groupie Tracker - All Artists",
		Artists: artists,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		return
	}
}

func handleNotFoundRoute(w http.ResponseWriter, r *http.Request, errorTmpl *template.Template) {
	handleError(w, errorTmpl, 404, "Page not found")
}

func handleError(w http.ResponseWriter, tmpl *template.Template, code int, message string) {
	w.WriteHeader(code)
	data := struct {
		Title   string
		Code    int
		Message string
	}{
		Title:   "Error",
		Code:    code,
		Message: message,
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, message, code)
	}
}

func fetchAllArtists() ([]Artist, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/artists")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return artists, nil
}

func fetchAllLocations() (map[int][]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/locations")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all locations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var all AllLocationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("failed to decode locations JSON: %w", err)
	}

	result := make(map[int][]string)
	for _, entry := range all.Index {
		result[entry.ID] = entry.Locations
	}
	return result, nil
}

func fetchArtistByID(id int) (*Artist, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	u := fmt.Sprintf("%s/artists/%d", baseURL, id)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var artist Artist
	if err := json.NewDecoder(resp.Body).Decode(&artist); err != nil {
		return nil, fmt.Errorf("failed to decode artist JSON: %w", err)
	}
	return &artist, nil
}

func fetchLocations(id int) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	u := fmt.Sprintf("%s/locations/%d", baseURL, id)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch locations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var ld LocationData
	if err := json.NewDecoder(resp.Body).Decode(&ld); err != nil {
		return nil, fmt.Errorf("failed to decode locations JSON: %w", err)
	}
	return ld.Locations, nil
}

func fetchDates(id int) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	u := fmt.Sprintf("%s/dates/%d", baseURL, id)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var cd ConcertDates
	if err := json.NewDecoder(resp.Body).Decode(&cd); err != nil {
		return nil, fmt.Errorf("failed to decode dates JSON: %w", err)
	}
	return cd.Dates, nil
}

func fetchRelations(id int) (map[string][]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	u := fmt.Sprintf("%s/relation/%d", baseURL, id)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var rd RelationData
	if err := json.NewDecoder(resp.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("failed to decode relations JSON: %w", err)
	}
	return rd.DatesLocations, nil
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(q) < 1 {
		w.Write([]byte("[]"))
		return
	}

	artists, err := fetchAllArtists()
	if err != nil {
		http.Error(w, "failed to fetch artists", http.StatusInternalServerError)
		return
	}

	allLocations, _ := fetchAllLocations()

	var suggestions []SearchSuggestion
	seen := make(map[string]bool)

	addSuggestion := func(label, sType string, id int) {
		key := fmt.Sprintf("%s|%s|%d", strings.ToLower(label), sType, id)
		if seen[key] {
			return
		}
		seen[key] = true
		suggestions = append(suggestions, SearchSuggestion{
			Label:    label,
			Type:     sType,
			ArtistID: id,
		})
	}

	for _, artist := range artists {
		if len(suggestions) >= 15 {
			break
		}

		if strings.Contains(strings.ToLower(artist.Name), q) {
			addSuggestion(artist.Name, "artist/band", artist.ID)
		}

		for _, member := range artist.Members {
			if strings.Contains(strings.ToLower(member), q) {
				addSuggestion(member, "member", artist.ID)
			}
		}

		if strings.Contains(strings.ToLower(artist.FirstAlbum), q) {
			addSuggestion(artist.FirstAlbum, "first album", artist.ID)
		}

		creationStr := fmt.Sprintf("%d", artist.CreationDate)
		if strings.Contains(creationStr, q) {
			addSuggestion(creationStr, "creation date", artist.ID)
		}

		if locs, ok := allLocations[artist.ID]; ok {
			for _, loc := range locs {
				if strings.Contains(strings.ToLower(loc), q) {
					addSuggestion(loc, "location", artist.ID)
				}
			}
		}
	}

	if suggestions == nil {
		suggestions = []SearchSuggestion{}
	}

	if err := json.NewEncoder(w).Encode(suggestions); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func handleArtistDetail(w http.ResponseWriter, r *http.Request, tmpl *template.Template, errorTmpl *template.Template) {
	log.Printf("/artist request: %s", r.URL.Path)

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		path := strings.TrimSuffix(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			idStr = parts[2]
		}
	}
	if idStr == "" {
		handleError(w, errorTmpl, 400, "Missing artist id")
		return
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		decoded, _ := url.PathUnescape(idStr)
		normalize := func(s string) string {
			s = strings.ToLower(s)
			return strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					return r
				}
				return -1
			}, s)
		}
		target := normalize(decoded)
		target = strings.TrimPrefix(target, "the")

		artists, err := fetchAllArtists()
		if err != nil {
			handleError(w, errorTmpl, 500, "Error fetching data from API")
			return
		}

		found := false
		for _, a := range artists {
			n := normalize(a.Name)
			n = strings.TrimPrefix(n, "the")
			if n == target || strings.Contains(n, target) || strings.Contains(target, n) {
				id = a.ID
				found = true
				break
			}
		}
		if !found {
			handleError(w, errorTmpl, 400, "Invalid artist id")
			return
		}
	}

	artist, err := fetchArtistByID(id)
	if err != nil {
		handleError(w, errorTmpl, 404, "Artist not found")
		return
	}

	locations, _ := fetchLocations(id)
	dates, _ := fetchDates(id)
	relationsMap, _ := fetchRelations(id)

	rels := make([]Relation, 0, len(relationsMap))
	for loc, ds := range relationsMap {
		rels = append(rels, Relation{Location: loc, Dates: ds})
	}

	ext := ExtendedArtist{
		ID:           artist.ID,
		Image:        artist.Image,
		Name:         artist.Name,
		Members:      artist.Members,
		CreationDate: artist.CreationDate,
		FirstAlbum:   artist.FirstAlbum,
		ConcertDates: dates,
		Locations:    locations,
		Relations:    rels,
	}

	artists, _ := fetchAllArtists()
	queenImage := ""
	for _, a := range artists {
		if strings.ToLower(strings.TrimSpace(a.Name)) == "queen" && a.Image != "" {
			queenImage = a.Image
			break
		}
	}
	if strings.TrimSpace(ext.Image) == "" && queenImage != "" {
		ext.Image = queenImage
	}

	data := struct {
		Title  string
		Artist ExtendedArtist
	}{
		Title:  ext.Name,
		Artist: ext,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing artist template: %v", err)
		return
	}
}

func handleArtistJSON(w http.ResponseWriter, r *http.Request) {
	log.Printf("/api/artist request: %s", r.URL.Path)
	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var idStr string
	if len(parts) >= 4 && parts[3] != "" {
		idStr = parts[3]
	} else {
		idStr = r.URL.Query().Get("id")
	}
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		decoded, _ := url.PathUnescape(idStr)
		normalize := func(s string) string {
			s = strings.ToLower(s)
			return strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					return r
				}
				return -1
			}, s)
		}
		target := normalize(decoded)
		target = strings.TrimPrefix(target, "the")

		artists, err := fetchAllArtists()
		if err != nil {
			http.Error(w, "artist not found", http.StatusNotFound)
			return
		}
		found := false
		for _, a := range artists {
			n := normalize(a.Name)
			nSimple := strings.TrimPrefix(n, "the")
			if n == target || nSimple == target || strings.Contains(n, target) || strings.Contains(target, nSimple) {
				id = a.ID
				found = true
				break
			}
		}
		if !found {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
	}

	artist, err := fetchArtistByID(id)
	if err != nil {
		http.Error(w, "artist not found", http.StatusNotFound)
		return
	}

	locations, _ := fetchLocations(id)
	dates, _ := fetchDates(id)
	relationsMap, _ := fetchRelations(id)

	rels := make([]Relation, 0, len(relationsMap))
	for loc, ds := range relationsMap {
		rels = append(rels, Relation{Location: loc, Dates: ds})
	}

	ext := ExtendedArtist{
		ID:           artist.ID,
		Image:        artist.Image,
		Name:         artist.Name,
		Members:      artist.Members,
		CreationDate: artist.CreationDate,
		FirstAlbum:   artist.FirstAlbum,
		ConcertDates: dates,
		Locations:    locations,
		Relations:    rels,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ext); err != nil {
		http.Error(w, "failed to encode json", http.StatusInternalServerError)
		return
	}
}
