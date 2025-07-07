package filestorage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	// "path/filepath" // Para manipulação de nomes de arquivo, se necessário
	// "github.com/google/uuid" // Para gerar nomes de arquivo únicos

	"cloud.google.com/go/storage"
	// "google.golang.org/api/option" // Para credenciais explícitas, se necessário
)

var (
	gcsClient     *storage.Client
	gcsBucketName string
	gcsProjectID  string
)

// FileStorageProvider defines an interface for file storage operations.
// This allows for easier testing and potential swapping of storage providers.
type FileStorageProvider interface {
	UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (fileURL string, err error)
	// DeleteFile(ctx context.Context, fileURL string) error // Opcional
}

// GCSStorageProvider implements FileStorageProvider using Google Cloud Storage.
type GCSStorageProvider struct {
	client     *storage.Client
	bucketName string
}

// InitializeGCSProvider initializes the Google Cloud Storage client and configuration.
func InitializeGCSProvider() (*GCSStorageProvider, error) {
	ctx := context.Background()

	gcsProjectID = os.Getenv("GCS_PROJECT_ID")
	gcsBucketName = os.Getenv("GCS_BUCKET_NAME")
	// GOOGLE_APPLICATION_CREDENTIALS é lido automaticamente pela biblioteca cliente se estiver definido.

	if gcsProjectID == "" {
		log.Println("GCS_PROJECT_ID not set. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}
	if gcsBucketName == "" {
		log.Println("GCS_BUCKET_NAME not set. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}

	var err error
	gcsClient, err = storage.NewClient(ctx) // Tenta usar credenciais do ambiente
	// Se GOOGLE_APPLICATION_CREDENTIALS estiver definido e válido, será usado.
	// Em um ambiente GCP (Cloud Run, GKE, etc.), as credenciais da conta de serviço associada são usadas.

	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Storage client: %w. Ensure GOOGLE_APPLICATION_CREDENTIALS is set correctly for local/Docker or Workload Identity is configured in GCP.", err)
	}

	log.Printf("Google Cloud Storage provider initialized for project %s, bucket %s", gcsProjectID, gcsBucketName)

	provider := &GCSStorageProvider{
		client:     gcsClient,
		bucketName: gcsBucketName,
	}
	// Atribuir o provider a uma variável global ou retorná-lo para ser injetado onde necessário.
	// Por enquanto, vamos apenas logar e a instanciação real será feita quando for usado.
	// Ou podemos ter uma variável global `DefaultFileStorageProvider FileStorageProvider = provider`

	return provider, nil
}

// UploadFile carrega um arquivo para o GCS e retorna sua URL pública.
// objectName deve ser o nome final do arquivo como você quer que apareça no bucket (ex: incluindo prefixos de pasta).
func (g *GCSStorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if g.client == nil || g.bucketName == "" {
		return "", fmt.Errorf("GCS provider not initialized or configured correctly")
	}

	bucket := g.client.Bucket(g.bucketName)
	obj := bucket.Object(objectName) // objectName já inclui o path completo dentro do bucket

	// Configurar o writer para o objeto GCS
	wc := obj.NewWriter(ctx)
	// Opcional: Definir ACL para tornar o objeto público para leitura
	// Isso depende da política do bucket. Se o bucket for público, isso pode não ser necessário.
	// Se o bucket for privado, você precisaria de URLs assinadas ou ACLs.
	// Para simplificar, vamos assumir que o objeto será publicamente legível.
	// Em produção, URLs assinadas são geralmente mais seguras.
	wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	// Opcional: Definir o tipo de conteúdo (MIME type) se conhecido
	// wc.ContentType = "image/jpeg" // Exemplo

	if _, err := io.Copy(wc, fileContent); err != nil {
		return "", fmt.Errorf("failed to copy file content to GCS object writer: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS object writer: %w", err)
	}

	// Construir a URL pública do objeto
	// Formato: https://storage.googleapis.com/[BUCKET_NAME]/[OBJECT_NAME]
	// Ou, se usar um domínio customizado: https://[CUSTOM_DOMAIN]/[OBJECT_NAME]
	// Por simplicidade, usaremos a URL padrão do storage.googleapis.com.
	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucketName, objectName)

	log.Printf("File uploaded successfully to GCS: %s", publicURL)
	return publicURL, nil
}

// DeleteFile (opcional, não implementado neste escopo inicial)
func (g *GCSStorageProvider) DeleteFile(ctx context.Context, fileURL string) error {
	// Para implementar: parsear fileURL para obter bucket e object name, depois chamar obj.Delete(ctx)
	return fmt.Errorf("GCS DeleteFile not yet implemented")
}

// Global instance of the file storage provider
var DefaultFileStorageProvider FileStorageProvider

// InitFileStorage initializes the default file storage provider based on configuration.
// Por enquanto, apenas GCS. No futuro, poderia ler uma config para escolher entre GCS, S3, local, etc.
func InitFileStorage() error {
	// Tentar inicializar GCS
	gcsProvider, err := InitializeGCSProvider()
	if err != nil {
		// Se a inicialização do GCS falhar mas não for fatal (ex: credenciais não configuradas mas não queremos que a app pare)
		// podemos logar o erro e continuar sem um provedor de arquivos funcional.
		log.Printf("Could not initialize GCS provider: %v. File uploads will not work.", err)
		// DefaultFileStorageProvider permanecerá nil. Os handlers precisarão verificar isso.
		return nil // Não retornar erro aqui para permitir que a app inicie mesmo sem GCS configurado.
	}

	if gcsProvider != nil {
		DefaultFileStorageProvider = gcsProvider
		log.Println("Default file storage provider set to GCS.")
	} else {
		log.Println("No file storage provider initialized (GCS config missing or invalid). File uploads will be disabled.")
	}
	return nil
}
