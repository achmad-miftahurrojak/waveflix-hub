

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	TaskTypeTMDBAPI    = "tmdb_api"
	TaskTypeDataProcess = "data_process"
	TaskTypeEmailSend   = "email_send"
)

var tmdbBaseUrl = "https://api.themoviedb.org/3"

var tmdbApiKey = ""

func getTmdbApiKey() string {
	if tmdbApiKey != "" {
		return tmdbApiKey
	}
	tmdbApiKey = os.Getenv("TMDB_API_KEY")
	return tmdbApiKey
}

func getMediaParam(q url.Values) string {
	return normalizeMedia(q.Get("media"))
}

func getTmdbLang(q url.Values) string {
	if lang := q.Get("language"); lang != "" {
		return lang
	}
	return "id-ID"
}

func getImageLangs(q url.Values) string {
	if lang := q.Get("include_image_language"); lang != "" {
		return lang
	}
	return "id,null"
}

func fetchJSON(targetURL string, mediaType string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("[tmdb] fetch error (%s): %w", mediaType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[tmdb] non-200 response (%s): %d", mediaType, resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("[tmdb] decode error (%s): %w", mediaType, err)
	}

	return result, nil
}

func sanitizeTMDBData(data map[string]interface{}, mediaType string) {

	delete(data, "adult")
	delete(data, "belongs_to_collection") 
}

func checkVidlinkAvailability(media, id string) bool {
	url := fmt.Sprintf("https://vidlink.pro/api/b/%s/%s", media, id)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func checkMajorProvider(media, id string) bool {

	return false
}
