package filestorage

import (
	"context"
	"fmt"
	"io"
	// "os" // Removido, config.Cfg é usado
	"phoenixgrc/backend/pkg/config" // Adicionado para acessar config.Cfg
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                   // Importar zap
	"time"

	// "path/filepath" // Para manipulação de nomes de arquivo, se necessário
	// "github.com/google/uuid" // Para gerar nomes de arquivo únicos

	"cloud.google.com/go/storage"
	// "google.golang.org/api/option" // Para credenciais explícitas, se necessário
)

// GCSStorageProvider implements FileStorageProvider using Google Cloud Storage.
// Variáveis globais gcsClient e gcsBucketName não são mais necessárias aqui.
type GCSStorageProvider struct {
	client     *storage.Client
	bucketName string
}

// InitializeGCSProvider initializes the Google Cloud Storage client and configuration.
func InitializeGCSProvider() (*GCSStorageProvider, error) {
	ctx := context.Background()

	// Usa config.Cfg para obter as configurações em vez de variáveis de ambiente diretas ou globais do pacote
	projectID := config.Cfg.GCSProjectID
	bucketName := config.Cfg.GCSBucketName

	// GOOGLE_APPLICATION_CREDENTIALS é lido automaticamente pela biblioteca cliente se estiver definido no ambiente.

	if projectID == "" {
		phxlog.L.Warn("GCS_PROJECT_ID not set in config. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}
	if bucketName == "" {
		phxlog.L.Warn("GCS_BUCKET_NAME not set in config. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}

	var err error
	localStorageClient, err := storage.NewClient(ctx) // Renomeado para localStorageClient para evitar confusão com globais (que foram removidas)
	// Se GOOGLE_APPLICATION_CREDENTIALS estiver definido e válido, será usado.
	// Em um ambiente GCP (Cloud Run, GKE, etc.), as credenciais da conta de serviço associada são usadas.

	if err != nil {
		phxlog.L.Error("Failed to create Google Cloud Storage client. Ensure GOOGLE_APPLICATION_CREDENTIALS is set correctly for local/Docker or Workload Identity is configured in GCP.", zap.Error(err))
		return nil, fmt.Errorf("failed to create Google Cloud Storage client: %w", err)
	}

	phxlog.L.Info("Google Cloud Storage provider initialized", zap.String("projectID", projectID), zap.String("bucketName", bucketName))

	provider := &GCSStorageProvider{
		client:     localStorageClient, // Usa o cliente localmente definido
		bucketName: bucketName,       // Usa o bucketName localmente definido
	}
	// Atribuir o provider a uma variável global ou retorná-lo para ser injetado onde necessário.
	// Por enquanto, vamos apenas logar e a instanciação real será feita quando for usado.
	// Ou podemos ter uma variável global `DefaultFileStorageProvider FileStorageProvider = provider`

	return provider, nil
}

// UploadFile carrega um arquivo para o GCS e retorna seu objectName (path no bucket).
// objectName deve ser o nome final do arquivo como você quer que apareça no bucket (ex: incluindo prefixos de pasta).
func (g *GCSStorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if g.client == nil || g.bucketName == "" {
		return "", fmt.Errorf("GCS provider not initialized or configured correctly")
	}

	bucket := g.client.Bucket(g.bucketName)
	obj := bucket.Object(objectName)

	wc := obj.NewWriter(ctx)
	// Objetos não são públicos por padrão. Acesso via GetSignedURL.
	// wc.ContentType = "image/jpeg" // Opcional: Definir o tipo de conteúdo

	if _, err := io.Copy(wc, fileContent); err != nil {
		return "", fmt.Errorf("failed to copy file content to GCS object writer: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS object writer: %w", err)
	}

	phxlog.L.Info("File uploaded successfully to GCS",
		zap.String("bucket", g.bucketName),
		zap.String("objectName", objectName))
	return objectName, nil // Retorna o objectName
}

// DeleteFile remove um arquivo do GCS usando o objectName (path no bucket).
func (g *GCSStorageProvider) DeleteFile(ctx context.Context, objectName string) error {
	if g.client == nil || g.bucketName == "" {
		return fmt.Errorf("GCS provider not initialized or configured correctly")
	}
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty for DeleteFile")
	}

	obj := g.client.Bucket(g.bucketName).Object(objectName)
	if err := obj.Delete(ctx); err != nil {
		if err == storage.ErrObjectNotExist {
			phxlog.L.Info("GCS DeleteFile: Object not found (considered successful for idempotency)",
				zap.String("objectName", objectName),
				zap.String("bucket", g.bucketName))
			return nil
		}
		phxlog.L.Error("Failed to delete object from GCS",
			zap.String("objectName", objectName),
			zap.String("bucket", g.bucketName),
			zap.Error(err))
		return fmt.Errorf("failed to delete object '%s' from GCS bucket '%s': %w", objectName, g.bucketName, err)
	}

	phxlog.L.Info("File deleted successfully from GCS",
		zap.String("bucket", g.bucketName),
		zap.String("objectName", objectName))
	return nil
}

// GetSignedURL gera uma URL assinada para um objeto no GCS.
func (g *GCSStorageProvider) GetSignedURL(ctx context.Context, objectName string, durationMinutes int) (string, error) {
	if g.client == nil || g.bucketName == "" {
		return "", fmt.Errorf("GCS provider not initialized or configured correctly")
	}
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty for GetSignedURL")
	}

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(time.Duration(durationMinutes) * time.Minute),
		// TODO: Considerar se o service account para assinar URLs precisa ser configurado explicitamente
		//       ou se as credenciais padrão do ambiente (ADC) são suficientes.
		//       Pode ser necessário: GoogleAccessID: "your-service-account-email", PrivateKey: []byte("your-private-key"),
	}

	signedURL, err := storage.SignedURL(g.bucketName, objectName, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for GCS object '%s': %w", objectName, err)
	}

	return signedURL, nil
}


// Note: DefaultFileStorageProvider and InitFileStorage are now in filestorage.go
