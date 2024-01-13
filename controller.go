package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func getURL(w http.ResponseWriter, r *http.Request) {
	// Extract space and ID parameters from the URL path
	vars := mux.Vars(r)
	space := vars["space"]
	id := vars["id"]

	// Reinitialize the database for the specified space
	outputDir := getOutputDirFromArgs()
	dbFile := filepath.Join(outputDir, space+".db")

	// Close the existing database connection if it's open
	if db != nil {
		db.Close()
	}

	// Reinitialize the SQLite database with the specified space
	initDB(dbFile)

	// Query the database for the short URL
	row := db.QueryRow(`
		SELECT id, url, meta, visited, last_visited_at, created_at, updated_at
		FROM short_urls
		WHERE id = ?
	`, id)

	// Parse the query result
	var shortURL ShortURL
	var lastVisitedAt sql.NullTime
	err := row.Scan(
		&shortURL.ID,
		&shortURL.URL,
		&shortURL.Meta,
		&shortURL.Visited,
		&lastVisitedAt,
		&shortURL.CreatedAt,
		&shortURL.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	// Assign the value to shortURL.LastVisitedAt only if it's not NULL
	if lastVisitedAt.Valid {
		shortURL.LastVisitedAt = lastVisitedAt.Time
	}

	// Respond with the short URL and scraped metadata (json)
	response := JSONResponse{Status: "success", Message: "Short URL found successfully", Data: &shortURL}
	respondJSON(w, http.StatusOK, response)
}

func getURLs(w http.ResponseWriter, r *http.Request) {
	// Extract space parameter from the URL path
	vars := mux.Vars(r)
	space := vars["space"]

	// Reinitialize the database for the specified space
	outputDir := getOutputDirFromArgs()
	dbFile := filepath.Join(outputDir, space+".db")

	// Close the existing database connection if it's open
	if db != nil {
		db.Close()
	}

	// Reinitialize the SQLite database with the specified space
	initDB(dbFile)

	// Parse query parameters for pagination
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("perPage"))
	if err != nil || perPage < 1 {
		perPage = 10
	}

	// Calculate the offset based on the page and perPage values
	offset := (page - 1) * perPage

	// Query the database to get paginated URLs
	rows, err := db.Query(`
		SELECT id, url, meta, visited, last_visited_at, created_at, updated_at
		FROM short_urls
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, perPage, offset)
	if err != nil {
		log.Println("Error querying short URLs:", err)
		http.Error(w, "Failed to get short URLs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Iterate over the rows and build the response
	var shortURLs []ShortURL
	for rows.Next() {
		var shortURL ShortURL
		var lastVisitedAt sql.NullTime
	
		err := rows.Scan(
			&shortURL.ID,
			&shortURL.URL,
			&shortURL.Meta,
			&shortURL.Visited,
			&lastVisitedAt,
			&shortURL.CreatedAt,
			&shortURL.UpdatedAt,
		)
		if err != nil {
			log.Println("Error scanning short URL row:", err)
			http.Error(w, "Failed to get short URLs", http.StatusInternalServerError)
			return
		}
	
		// Assign the value to shortURL.LastVisitedAt only if it's not NULL
		if lastVisitedAt.Valid {
			shortURL.LastVisitedAt = lastVisitedAt.Time
		}

		shortURL.Path = "/" + space + "/" + shortURL.ID
		shortURLs = append(shortURLs, shortURL)
	}

	// Respond with the paginated short URLs
	responseJSON, err := json.Marshal(shortURLs)
	if err != nil {
		log.Println("Error encoding short URLs to JSON:", err)
		http.Error(w, "Failed to encode short URLs", http.StatusInternalServerError)
		return
	}

	/* if responseJSON is null then response array is empty */
	if len(shortURLs) == 0 {
		responseJSON = []byte("[]")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func deleteURL(w http.ResponseWriter, r *http.Request) {
	// Extract space and ID parameters from the URL path
	vars := mux.Vars(r)
	space := vars["space"]
	id := vars["id"]

	// Reinitialize the database for the specified space
	outputDir := getOutputDirFromArgs()
	dbFile := filepath.Join(outputDir, space+".db")

	// Close the existing database connection if it's open
	if db != nil {
		db.Close()
	}

	// Reinitialize the SQLite database with the specified space
	initDB(dbFile)

	// Check if the URL with the given ID exists
	if !idExists(id) {

		// Respond with a success message
		response := JSONResponse{Status: "error", Message: "URL not found"}
		respondJSON(w, http.StatusNotFound, response)
		return
	}

	// Delete the URL from the database
	_, err := db.Exec("DELETE FROM short_urls WHERE id = ?", id)
	if err != nil {
		log.Println("Error deleting short URL:", err)
		http.Error(w, "Failed to delete short URL", http.StatusInternalServerError)
		return
	}


	// Respond with a success message
	response := JSONResponse{Status: "success", Message: "URL deleted successfully"}
	respondJSON(w, http.StatusOK, response)
}


func redirectURL(w http.ResponseWriter, r *http.Request) {
	// Extract space and ID parameters from the URL path
	vars := mux.Vars(r)
	space := vars["space"]
	id := vars["id"]

	// Reinitialize the database for the specified space
	outputDir := getOutputDirFromArgs()
	dbFile := filepath.Join(outputDir, space+".db")

	// Close the existing database connection if it's open
	if db != nil {
		db.Close()
	}

	// Reinitialize the SQLite database with the specified space
	initDB(dbFile)

	// Get the long URL and meta information for the given ID and space
	var longURL string
	var meta string
	err := db.QueryRow("SELECT url, meta FROM short_urls WHERE id = ?", id).Scan(&longURL, &meta)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Increment the 'visited' count and update 'last_visited_at'
	_, err = db.Exec("UPDATE short_urls SET visited = visited + 1, last_visited_at = ? WHERE id = ?", time.Now(), id)
	if err != nil {
		log.Println("Error updating statistics:", err)
		// Continue with the redirection even if there's an error updating statistics
	}

	// Redirect to the long URL
	http.Redirect(w, r, longURL, http.StatusFound)
}

func createShortURLWithScrape(w http.ResponseWriter, r *http.Request) {
	// Initialize SQLite database with the current year's hash
	outputDir := getOutputDirFromArgs()
	currentYear := getHashedYear(time.Now().Year())
	dbFile := filepath.Join(outputDir, currentYear+".db")

	// Close the existing database connection if it's open
	if db != nil {
		db.Close()
	}

	// Initialize a new SQLite database with the current year's hash
	initDB(dbFile)

	space := currentYear

	// Parse the request body as JSON
	var requestData struct {
		URL      string            `json:"url"`
		CustomID string            `json:"custom_id"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if requestData.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Use custom ID if provided, generate a new one otherwise
	var id string
	if requestData.CustomID != "" {
		// Check if the custom ID is available in the database
		if idExists(requestData.CustomID) {
			http.Error(w, "Custom ID is already in use", http.StatusBadRequest)
			return
		}
		id = requestData.CustomID
	} else {
		// Generate a unique ID (hash of current time)
		id = generateUniqueID(space, requestData.URL)
	}

	// Scrape metadata from the provided URL
	meta, err := scrapePageMetadata(requestData.URL)
	if err != nil {
		log.Println("Error scraping metadata:", err)
	}

	fmt.Println("Scraped metadata:", meta)

	// Convert the meta map to a MetaScanner
	metaScanner := ConvertPageMetadataToMetaScanner(meta)

	// Convert the meta map to a JSON string
	metaJSON, err := json.Marshal(metaScanner)
	if err != nil {
		http.Error(w, "Failed to convert meta to JSON string", http.StatusInternalServerError)
		return
	}

	// Insert the short URL into the database with the parsed meta values
	createdAt := time.Now()
	_, err = db.Exec(`
		INSERT INTO short_urls (id, url, meta, visited, last_visited_at, created_at, updated_at)
		VALUES (?, ?, ?, 0, NULL, ?, ?)
	`, id, requestData.URL, string(metaJSON), createdAt, createdAt)
	
	if err != nil {
		log.Println("Error inserting short URL into the database:", err)
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	// Respond with the short URL and scraped metadata (json)
	data := ShortURL{
		ID:            id,
		URL:           requestData.URL,
		Meta:          metaScanner, // Use the converted MetaScanner
		Path:          "/" + space + "/" + id,
		LastVisitedAt: time.Time{},
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}

	response := JSONResponse{Status: "success", Message: "Short URL created successfully", Data: &data}
	respondJSON(w, http.StatusCreated, response)
}
