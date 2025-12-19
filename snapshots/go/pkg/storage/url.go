package storage

import (
	"log"
	"net/url"
	"strings"
)

func transformURL(storageURL string) string {
	u, err := url.Parse(storageURL)
	if err != nil {
		return storageURL
	}

	// Backwards compatibility:
	// In prior versions, "gcs://" was used instead of "gs://".
	if u.Scheme == "gcs" {
		log.Printf("WARNING: 'gcs://' is deprecated in favor of 'gs://', " +
			"and will be removed in the future. " +
			"Please update your storage URLs.")
		u.Scheme = "gs"
	}

	// For cloud storage URLs (s3:// and gs://),
	// the path component specifies a subdirectory prefix.
	// gocloud.dev ignores the path, so we convert it to a ?prefix= query parameter.
	if u.Scheme == "s3" || u.Scheme == "gs" {
		if path := strings.TrimPrefix(u.Path, "/"); path != "" {
			// Ensure prefix ends with "/" for proper subdirectory behavior.
			if !strings.HasSuffix(path, "/") {
				path += "/"
			}

			q := u.Query()
			q.Set("prefix", path)
			u.RawQuery = q.Encode()
			u.Path = ""
		}
	}

	return u.String()
}
