package filestorage

import (
	"context"
	"io"
	"log"
	"phoenixgrc/backend/pkg/config"
)

// FileStorageProvider defines an interface for file storage operations.
type FileStorageProvider interface {
	// UploadFile uploads a file and returns the unique objectName (key/path) of the stored file.
	UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (storedObjectName string, err error)
	DeleteFile(ctx context.Context, objectName string) error // Mudado fileURL para objectName
	GetSignedURL(ctx context.Context, objectName string, durationMinutes int) (signedURL string, err error)
}

// DefaultFileStorageProvider holds the initialized default provider.
var DefaultFileStorageProvider FileStorageProvider

// InitFileStorage initializes the default file storage provider based on configuration.
func InitFileStorage() error {
	providerType := config.Cfg.FileStorageProvider
	log.Printf("Initializing file storage with provider type: %s", providerType)

	var err error
	switch providerType {
	case "s3":
		DefaultFileStorageProvider, err = InitializeS3Provider()
		if err != nil {
			log.Printf("Failed to initialize S3 storage provider: %v. Falling back if possible or disabling uploads.", err)
			// Potentially try GCS as a fallback or just disable. For now, let's be explicit.
			// If S3 init fails, DefaultFileStorageProvider will be nil if InitializeS3Provider returns nil on config error.
		}
		if DefaultFileStorageProvider == nil && err == nil { // S3 specifically not configured (e.g. missing bucket)
			log.Println("S3 provider not configured (e.g., missing bucket/region). File uploads via S3 disabled.")
		}
	case "gcs":
		DefaultFileStorageProvider, err = InitializeGCSProvider()
		if err != nil {
			log.Printf("Failed to initialize GCS storage provider: %v. Falling back if possible or disabling uploads.", err)
		}
		if DefaultFileStorageProvider == nil && err == nil { // GCS specifically not configured
			log.Println("GCS provider not configured (e.g., missing project/bucket). File uploads via GCS disabled.")
		}
	default:
		log.Printf("Unsupported FILE_STORAGE_PROVIDER: %s. File uploads will be disabled.", providerType)
		// DefaultFileStorageProvider will remain nil
	}

	if DefaultFileStorageProvider != nil {
		log.Printf("File storage provider '%s' initialized successfully.", providerType)
	} else {
		log.Printf("No file storage provider initialized. File uploads will be disabled.")
		// No error returned here to allow app to start, but uploads won't work.
	}
	return nil // Always return nil to not block app startup if storage isn't critical for all ops
}
