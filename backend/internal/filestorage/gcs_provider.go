package filestorage

import (
	"context"
	"fmt"
	"io"
	"log"
	// "os" // Removido, config.Cfg é usado
	"phoenixgrc/backend/pkg/config" // Adicionado para acessar config.Cfg

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
		log.Println("GCS_PROJECT_ID not set in config. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}
	if bucketName == "" {
		log.Println("GCS_BUCKET_NAME not set in config. File upload to GCS will be disabled.")
		return nil, nil // Retorna nil, nil para indicar que o provedor está desabilitado
	}

	var err error
	localStorageClient, err := storage.NewClient(ctx) // Renomeado para localStorageClient para evitar confusão com globais (que foram removidas)
	// Se GOOGLE_APPLICATION_CREDENTIALS estiver definido e válido, será usado.
	// Em um ambiente GCP (Cloud Run, GKE, etc.), as credenciais da conta de serviço associada são usadas.

	if err != nil {
		return nil, fmt.Errorf("failed to create Google Cloud Storage client: %w. Ensure GOOGLE_APPLICATION_CREDENTIALS is set correctly for local/Docker or Workload Identity is configured in GCP.", err)
	}

	log.Printf("Google Cloud Storage provider initialized for project %s, bucket %s", projectID, bucketName)

	provider := &GCSStorageProvider{
		client:     localStorageClient, // Usa o cliente localmente definido
		bucketName: bucketName,       // Usa o bucketName localmente definido
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

// Note: DefaultFileStorageProvider and InitFileStorage are now in filestorage.go
