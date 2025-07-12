package filestorage

import (
	"context"
	"io"
	"phoenixgrc/backend/pkg/config"
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
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
	phxlog.L.Info("Initializing file storage", zap.String("provider_type", providerType))

	var err error
	switch providerType {
	case "s3":
		DefaultFileStorageProvider, err = InitializeS3Provider()
		if err != nil {
			phxlog.L.Error("Failed to initialize S3 storage provider. File uploads via S3 may be disabled.", zap.Error(err))
			// Se err != nil, DefaultFileStorageProvider já será nil por InitializeS3Provider
		}
		if DefaultFileStorageProvider == nil && err == nil { // S3 especificamente não configurado (ex: bucket ausente)
			phxlog.L.Warn("S3 provider not configured (e.g., missing bucket/region). File uploads via S3 disabled.")
		}
	case "gcs":
		DefaultFileStorageProvider, err = InitializeGCSProvider()
		if err != nil {
			phxlog.L.Error("Failed to initialize GCS storage provider. File uploads via GCS may be disabled.", zap.Error(err))
		}
		if DefaultFileStorageProvider == nil && err == nil { // GCS especificamente não configurado
			phxlog.L.Warn("GCS provider not configured (e.g., missing project/bucket). File uploads via GCS disabled.")
		}
	default:
		phxlog.L.Warn("Unsupported FILE_STORAGE_PROVIDER. File uploads will be disabled.", zap.String("provider_type", providerType))
		// DefaultFileStorageProvider will remain nil
	}

	if DefaultFileStorageProvider != nil {
		phxlog.L.Info("File storage provider initialized successfully.", zap.String("provider_type", providerType))
	} else {
		phxlog.L.Warn("No file storage provider initialized. File uploads will be disabled.")
	}
	return nil // Always return nil to not block app startup if storage isn't critical for all ops
}
