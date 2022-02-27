package api

import (
	"github.com/itzg/restify"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
	"strings"
)

const (
	defaultCacheDuration = time.Hour
	downloadPage = "https://www.minecraft.net/en-us/download/server/bedrock"
)

var (
	cachedArchiveUrl = ""
	cachedArchiveUrl2 = ""
	cacheUntil       time.Time
)

func GetLatest(w http.ResponseWriter, r *http.Request) {
	if cachedArchiveUrl == "" || time.Now().After(cacheUntil) {
		var err *lookupError
		cachedArchiveUrl, err = lookupLatestVersionForLinux()
		if err !=  nil {
			log.Printf("E: %s", err)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(err.statusCode)
			w.Write([]byte(err.Error()))
			return
		}
		cachedArchiveUrl2, err = lookupLatestVersionForWindows()
		if err !=  nil {
			log.Printf("E: %s", err)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(err.statusCode)
			w.Write([]byte(err.Error()))
			return
		}
		cacheDuration := loadCacheDuration()
		cacheUntil = time.Now().Add(cacheDuration)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("linux="))
	w.Write([]byte(cachedArchiveUrl))
	w.Write([]byte("\nwindows="))
	w.Write([]byte(cachedArchiveUrl2))
}

func loadCacheDuration() time.Duration {
	var cacheDuration time.Duration
	cacheDurationStr := os.Getenv("CACHE_DURATION")
	if cacheDurationStr != "" {
		var parseErr error
		cacheDuration, parseErr = time.ParseDuration(cacheDurationStr)
		if parseErr != nil {
			cacheDurationStr = ""
		}
	}
	if cacheDurationStr == "" {
		cacheDuration = defaultCacheDuration
	}
	return cacheDuration
}

type lookupError struct {
	message string
	wrapped error
	statusCode int
}

func newLookupError(message string, wrapped error, statusCode int) *lookupError {
	return &lookupError{message: message, wrapped: wrapped, statusCode: statusCode}
}

func (e *lookupError) Unwrap() error {
	return e.wrapped
}

func (e *lookupError) Error() string {
	return e.message
}

func lookupLatestVersionForLinux() (string, *lookupError) {
	downloadUrl, err := url.Parse(downloadPage)
	if err != nil {
		return "", newLookupError("Failed to parse download URL", err, http.StatusInternalServerError)
	}

	content, err := restify.LoadContent(downloadUrl, "mc-bds-helper/latest", restify.WithHeaders(
		map[string]string{
			"accept-language": "*",
		},
	))
	if err != nil {
		return "", newLookupError("Failed to load content", err, http.StatusInternalServerError)
	}

	subset := restify.FindSubsetByAttributeNameValue(content, "data-platform", "serverBedrockLinux")
	if len(subset) == 0 {
		return "", newLookupError("Failed to locate data-platform element", nil, http.StatusBadGateway)
	}

	for _, attribute := range subset[0].Attr {
		if attribute.Key == "href" {
			var result string
			var footIndex int
			result := attribute.Val
			footIndex := strings.Index(result,".zip")
			return result[42:footIndex], nil
		}
	}

	return "", newLookupError("Matched element was missing href", nil, http.StatusBadGateway)
}

func lookupLatestVersionForWindows() (string, *lookupError) {
	downloadUrl, err := url.Parse(downloadPage)
	if err != nil {
		return "", newLookupError("Failed to parse download URL", err, http.StatusInternalServerError)
	}

	content, err := restify.LoadContent(downloadUrl, "mc-bds-helper/latest", restify.WithHeaders(
		map[string]string{
			"accept-language": "*",
		},
	))
	if err != nil {
		return "", newLookupError("Failed to load content", err, http.StatusInternalServerError)
	}

	subset := restify.FindSubsetByAttributeNameValue(content, "data-platform", "serverBedrockWindows")
	if len(subset) == 0 {
		return "", newLookupError("Failed to locate data-platform element", nil, http.StatusBadGateway)
	}

	for _, attribute := range subset[0].Attr {
		if attribute.Key == "href" {
			var result string
			var footIndex int
			result := attribute.Val
			footIndex := strings.Index(result,".zip")
			return result[40:footIndex], nil
		}
	}

	return "", newLookupError("Matched element was missing href", nil, http.StatusBadGateway)
}
