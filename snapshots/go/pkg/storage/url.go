package storage

import "net/url"

func backwardsCompatibleStorageURL(storageURL string) string {
	u, err := url.Parse(storageURL)
	if err != nil {
		return storageURL
	}

	// Backwards compatibility:
	// In prior versions, "gcs://" was used instead of "gs://".
	if u.Scheme == "gcs" {
		u.Scheme = "gs"
	}

	return u.String()
}
