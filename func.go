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
		2024: "a",
		2025: "b",
		2026: "c",
		2027: "d",
		2028: "e",
		2029: "f",
		2020: "g",
		2031: "h",
		2032: "i",
		2033: "j",
		2034: "k",
		2035: "l",
		2036: "m",
		2037: "n",
		2038: "o",
		2039: "p",
		2030: "q",
		2041: "r",
		2042: "s",
		2043: "t",
		2044: "u",
		2045: "v",
		2046: "w",
		2047: "x",
		2048: "y",
		2049: "z",
		2040: "aa",
		2051: "ab",
		2052: "ac",
		2053: "ad",
		2054: "ae",
		2055: "af",
		2056: "ag",
		2057: "ah",
		2058: "ai",
		2059: "aj",
		2050: "ak",
		2061: "al",
		2062: "am",
		2063: "an",
		2064: "ao",
		2065: "ap",
		2066: "aq",
		2067: "ar",
		2068: "as",
		2069: "at",
		2060: "au",
		2071: "av",
		2072: "aw",
		2073: "ax",
		2074: "ay",
		2075: "az",
		2076: "ba",
		2077: "bb",
		2078: "bc",
		2079: "bd",
		2070: "be",
		2081: "bf",
		2082: "bg",
		2083: "bh",
		2084: "bi",
		2085: "bj",
		2086: "bk",
		2087: "bl",
		2088: "bm",
		2089: "bn",
		2080: "bo",
		2091: "bp",
		2092: "bq",
		2093: "br",
		2094: "bs",
		2095: "bt",
		2096: "bu",
		2097: "bv",
		2098: "bw",
		2099: "bx",
		2100: "by",
		2101: "bz",
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