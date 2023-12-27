package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type ShortURL struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	Meta          MetaScanner       `json:"meta"`
	Visited       int               `json:"visited"`
	LastVisitedAt time.Time         `json:"last_visited_at"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// MetaScanner is a custom type that implements the sql.Scanner interface
type MetaScanner map[string]string

// Scan implements the sql.Scanner interface for MetaScanner
func (m *MetaScanner) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	// Assuming that the value is a JSON-encoded string
	var jsonString string
	switch v := value.(type) {
	case []byte:
		jsonString = string(v)
	case string:
		jsonString = v
	default:
		return fmt.Errorf("unsupported type for MetaScanner: %T", value)
	}

	// Parse the JSON-encoded string into a map
	var meta map[string]string
	if err := json.Unmarshal([]byte(jsonString), &meta); err != nil {
		return err
	}

	*m = meta
	return nil
}



var db *sql.DB
var outputDir = flag.String("output", "./output", "Output directory for the database files")

func main() {
	r := mux.NewRouter()

	// Define command-line flags
	port := flag.Int("port", 8080, "Port number for the server")
	flag.Parse()


	fmt.Print(*outputDir);
	// API Routes
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(apiKeyMiddleware)

	apiRouter.HandleFunc("", createShortURL).Methods("POST")
	apiRouter.HandleFunc("/{space}", getURLs).Methods("GET")
	apiRouter.HandleFunc("/{space}/{id}", deleteURL).Methods("DELETE")


	// Redirect Route
	r.HandleFunc("/{space}/{id}", redirectURL).Methods("GET")

	// Start the server
	portStr := fmt.Sprintf(":%d", *port)
	log.Printf("Server started on port %s\n", portStr)
	log.Fatal(http.ListenAndServe(portStr, r))

}

func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("GO_SHORT_KEY")
		if apiKey == "" {
			http.Error(w, "API key is not set", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("X-API-Key") != apiKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func createShortURL(w http.ResponseWriter, r *http.Request) {
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
		URL     string            `json:"url"`
		CustomID string          `json:"custom_id"`
		Meta    map[string]string `json:"meta"`
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

	// Convert the meta map to a JSON string
	metaJSON, err := json.Marshal(requestData.Meta)
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

	// Respond with the short URL
	shortURL := fmt.Sprintf("/%s/%s", space, id)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
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
	
		shortURLs = append(shortURLs, shortURL)
	}

	// Respond with the paginated short URLs
	responseJSON, err := json.Marshal(shortURLs)
	if err != nil {
		log.Println("Error encoding short URLs to JSON:", err)
		http.Error(w, "Failed to encode short URLs", http.StatusInternalServerError)
		return
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

func getOutputDirFromArgs() string {
	outputDir := *outputDir
		
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.Mkdir(outputDir, os.ModePerm)
	}

	return outputDir
}

func initDB(dbFile string) {
	fmt.Println("Opening database:", dbFile)
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	// Create the table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS short_urls (
			id TEXT PRIMARY KEY,
			url TEXT,
			meta TEXT,
			visited INTEGER,
			last_visited_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}
