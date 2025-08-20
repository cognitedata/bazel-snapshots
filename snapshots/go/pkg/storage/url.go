package storage

import (
	"log"
	"net/url"
)

func backwardsCompatibleStorageURL(storageURL string) string {
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

	return u.String()
}
