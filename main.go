package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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
	Path		  string 			`json:"path"`
}

type JSONResponse struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    *ShortURL  `json:"data"`
	Results []ShortURL `json:"results"`
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

	apiRouter.HandleFunc("", createShortURLWithScrape).Methods("POST")
	apiRouter.HandleFunc("/{space}", getURLs).Methods("GET")
	apiRouter.HandleFunc("/{space}/{id}", deleteURL).Methods("DELETE")
	apiRouter.HandleFunc("/{space}/{id}", getURL).Methods("GET")


	// Redirect Route
	r.HandleFunc("/{space}/{id}", redirectURL).Methods("GET")

	// Start the server
	portStr := fmt.Sprintf(":%d", *port)
	log.Printf("Server started on http://localhost:%s/\n", portStr)
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

// func createShortURL(w http.ResponseWriter, r *http.Request) {
// 	// Initialize SQLite database with the current year's hash
// 	outputDir := getOutputDirFromArgs()
// 	currentYear := getHashedYear(time.Now().Year())
// 	dbFile := filepath.Join(outputDir, currentYear+".db")

// 	// Close the existing database connection if it's open
// 	if db != nil {
// 		db.Close()
// 	}

// 	// Initialize a new SQLite database with the current year's hash
// 	initDB(dbFile)
	
// 	space := currentYear

// 	// Parse the request body as JSON
// 	var requestData struct {
// 		URL     string            `json:"url"`
// 		CustomID string           `json:"custom_id"`
// 		Meta    map[string]string `json:"meta"`
// 	}

// 	decoder := json.NewDecoder(r.Body)
// 	if err := decoder.Decode(&requestData); err != nil {
// 		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
// 		return
// 	}

// 	// Validate input
// 	if requestData.URL == "" {
// 		http.Error(w, "URL is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Use custom ID if provided, generate a new one otherwise
// 	var id string
// 	if requestData.CustomID != "" {
// 		// Check if the custom ID is available in the database
// 		if idExists(requestData.CustomID) {
// 			http.Error(w, "Custom ID is already in use", http.StatusBadRequest)
// 			return
// 		}
// 		id = requestData.CustomID
// 	} else {
// 		// Generate a unique ID (hash of current time)
// 		id = generateUniqueID(space, requestData.URL)
// 	}

// 	// Convert the meta map to a JSON string
// 	metaJSON, err := json.Marshal(requestData.Meta)
// 	if err != nil {
// 		http.Error(w, "Failed to convert meta to JSON string", http.StatusInternalServerError)
// 		return
// 	}

// 	// Insert the short URL into the database with the parsed meta values
// 	createdAt := time.Now()
// 	_, err = db.Exec(`
// 		INSERT INTO short_urls (id, url, meta, visited, last_visited_at, created_at, updated_at)
// 		VALUES (?, ?, ?, 0, NULL, ?, ?)
// 	`, id, requestData.URL, string(metaJSON), createdAt, createdAt)
// 	if err != nil {
// 		log.Println("Error inserting short URL into the database:", err)
// 		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
// 		return
// 	}

// 	// Respond with the short URL (json)
// 	data := ShortURL{
// 		ID:      id,
// 		URL:     requestData.URL,
// 		Meta:    requestData.Meta,
// 		Path:    "/"+space+"/"+id,
// 	}
// 	response := JSONResponse{Status: "success", Message: "Short URL created successfully", Data: &data}
// 	respondJSON(w, http.StatusCreated, response)
// }


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
