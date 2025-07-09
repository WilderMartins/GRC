package filestorage

import (
	"context"
	"fmt"
	"io"
	"log"
	"phoenixgrc/backend/pkg/config" // Referência ao config da aplicação

	"github.com/aws/aws-sdk-go-v2/aws"
	awsGoConfig "github.com/aws/aws-sdk-go-v2/config" // Alias para evitar conflito com pkg/config
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager" // Para S3 Upload Manager
	// "github.com/aws/smithy-go" // Para error handling mais específico, se necessário
)

// S3StorageProvider implements FileStorageProvider using Amazon S3.
type S3StorageProvider struct {
	client     *s3.Client
	uploader   *manager.Uploader // S3 Upload Manager para multipart uploads
	bucketName string
	region     string // Região do bucket S3
}

// InitializeS3Provider initializes the S3 client and configuration.
// Retorna nil, nil se o S3 não estiver configurado para não bloquear o início da app.
func InitializeS3Provider() (*S3StorageProvider, error) {
	bucket := config.Cfg.AWSS3Bucket
	region := config.Cfg.AWSRegion // Reutiliza a região configurada para SES, mas pode ser específica para S3

	if bucket == "" {
		log.Println("AWS_S3_BUCKET not set. File upload to S3 will be disabled.")
		return nil, nil
	}
	if region == "" {
		log.Println("AWS_REGION (for S3) not set. File upload to S3 will be disabled.")
		return nil, nil
	}

	// Carregar configuração AWS SDK (usa credenciais do ambiente: variáveis ou IAM role)
	// A sessão AWS já pode ter sido inicializada por SES (InitializeAWSSession em webhook_notifier.go)
	// Se não, precisamos garantir que seja carregada.
	// Para evitar dependência de ordem de inicialização, podemos carregar a config aqui também se necessário.
	// Reutilizando a lógica de InitializeAWSSession (se ela for global e segura para chamar múltiplas vezes)
	// ou duplicando a carga de config específica para S3.
	// Por simplicidade, vamos assumir que a config AWS será carregada globalmente ou aqui.

	sdkConfig, err := awsGoConfig.LoadDefaultConfig(context.TODO(), awsGoConfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config for S3: %w. Ensure AWS credentials and region are configured.", err)
	}
	log.Println("AWS SDK config loaded successfully for S3, region:", region)

	s3Client := s3.NewFromConfig(sdkConfig)
	uploader := manager.NewUploader(s3Client)


	log.Printf("Amazon S3 storage provider initialized for bucket %s, region %s", bucket, region)

	return &S3StorageProvider{
		client:     s3Client,
		uploader:   uploader,
		bucketName: bucket,
		region: region,
	}, nil
}

// UploadFile carrega um arquivo para o S3 e retorna sua URL.
// objectName deve ser o nome final do arquivo (chave) no bucket.
func (s *S3StorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if s.client == nil || s.uploader == nil || s.bucketName == "" {
		return "", fmt.Errorf("S3 provider not initialized or configured correctly")
	}

	// Opcional: Adicionar ContentType se puder ser determinado a partir do fileContent ou nome do arquivo.
	//contentType := "application/octet-stream" // Default
	// Se tivermos o header do arquivo original, poderíamos usar:
	// fileHeader, ok := fileContent.(*multipart.FileHeader)
	// if ok { contentType = fileHeader.Header.Get("Content-Type") }

	// O S3 Upload Manager lida com multipart uploads automaticamente para arquivos maiores.
	uploadOutput, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
		Body:   fileContent,
		// ACL: types.ObjectCannedACLPublicRead, // Para tornar o objeto publicamente legível
		// ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3 (bucket: %s, key: %s): %w", s.bucketName, objectName, err)
	}

	// A URL do objeto S3 pode ser construída de várias formas:
	// 1. Path-style: https://s3.<region>.amazonaws.com/<bucket>/<key>
	// 2. Virtual-hosted-style: https://<bucket>.s3.<region>.amazonaws.com/<key> (preferível)
	// uploadOutput.Location é a URL do objeto, geralmente no formato virtual-hosted.
	publicURL := uploadOutput.Location
	if publicURL == "" { // Fallback se Location não for preenchido (improvável com uploader)
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, objectName)
	}


	log.Printf("File uploaded successfully to S3: %s", publicURL)
	return publicURL, nil
}

// DeleteFile (opcional, pode ser implementado se necessário)
func (s *S3StorageProvider) DeleteFile(ctx context.Context, fileURL string) error {
	// Para implementar: parsear fileURL para obter bucket (verificar se é o s.bucketName) e object key.
	// Depois chamar s.client.DeleteObject(ctx, &s3.DeleteObjectInput{...})
	return fmt.Errorf("S3 DeleteFile not yet implemented")
}

// TODO: Adicionar lógica de deleção se o prompt exigir ou for uma melhoria futura.
// TODO: Considerar URLs assinadas para acesso privado em vez de ACLs públicas.
```
