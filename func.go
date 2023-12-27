package main // Check if the custom ID exists in the database for the specified space

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func idExists(id string) bool {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM short_urls WHERE id = ?
	`, id).Scan(&count)
	if err != nil {
		log.Println("Error checking if ID exists:", err)
		return true // Assume the ID exists to avoid using it
	}
	return count > 0
}

// Generate a unique ID based on the current time and space
func generateUniqueID(space string, url string) string {
	currentTimeNano := time.Now().UnixNano()
	// combine currentTime + url + space
	data := strconv.FormatInt(currentTimeNano, 10)+ url +space; 
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// get 5 last characters
	return hash[:5]
}

func getHashedYear(year int) string {
	hashData := map[int]string{
		2023: "a",
		2024: "b",
		2025: "c",
		2026: "d",
		2027: "e",
		2028: "f",
		2029: "g",
		2030: "h",
		2031: "i",
		2032: "j",
		2033: "k",
		2034: "l",
		2035: "m",
		2036: "n",
		2037: "o",
		2038: "p",
		2039: "q",
		2040: "r",
		2041: "s",
		2042: "t",
		2043: "u",
		2044: "v",
		2045: "w",
		2046: "x",
		2047: "y",
		2048: "z",
		2049: "aa",
		2050: "ab",
		2051: "ac",
		2052: "ad",
		2053: "ae",
		2054: "af",
		2055: "ag",
		2056: "ah",
		2057: "ai",
		2058: "aj",
		2059: "ak",
		2060: "al",
		2061: "am",
		2062: "an",
		2063: "ao",
		2064: "ap",
		2065: "aq",
		2066: "ar",
		2067: "as",
		2068: "at",
		2069: "au",
		2070: "av",
		2071: "aw",
		2072: "ax",
		2073: "ay",
		2074: "az",
		2075: "ba",
		2076: "bb",
		2077: "bc",
		2078: "bd",
		2079: "be",
		2080: "bf",
		2081: "bg",
		2082: "bh",
		2083: "bi",
		2084: "bj",
		2085: "bk",
		2086: "bl",
		2087: "bm",
		2088: "bn",
		2089: "bo",
		2090: "bp",
		2091: "bq",
		2092: "br",
		2093: "bs",
		2094: "bt",
		2095: "bu",
		2096: "bv",
		2097: "bw",
		2098: "bx",
		2099: "by",
		2100: "bz",
	}

	return hashData[year]
}

// respondJSON sends a JSON response with the specified status code and data
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	responseJSON, err := json.Marshal(data)
	if err != nil {
		log.Println("Error encoding JSON response:", err)
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseJSON)
}

type PageMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

func scrapePageMetadata(url string) (*PageMetadata, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	meta := &PageMetadata{}

	// Get title
	meta.Title = doc.Find("title").Text()

	// Get description
	doc.Find("meta[name=description]").Each(func(i int, s *goquery.Selection) {
		meta.Description, _ = s.Attr("content")
	})

	// Try to get og:image
	doc.Find("meta[property=og:image]").Each(func(i int, s *goquery.Selection) {
		meta.ImageURL, _ = s.Attr("content")
	})

	// If og:image is empty, try to get Twitter card image
	if meta.ImageURL == "" {
		doc.Find("meta[name='twitter:image']").Each(func(i int, s *goquery.Selection) {
			meta.ImageURL, _ = s.Attr("content")
		})
	}

	return meta, nil
}

// ConvertPageMetadataToMetaScanner converts *PageMetadata to MetaScanner
func ConvertPageMetadataToMetaScanner(pageMeta *PageMetadata) MetaScanner {
	metaScanner := make(MetaScanner)
	if pageMeta != nil {
		metaScanner["title"] = pageMeta.Title
		metaScanner["description"] = pageMeta.Description
		metaScanner["image"] = pageMeta.ImageURL
	}

	return metaScanner
}