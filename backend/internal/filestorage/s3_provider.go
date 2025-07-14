package filestorage

import (
	"context"
	"fmt"
	"io"
	"phoenixgrc/backend/pkg/config" // Referência ao config da aplicação
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                   // Importar zap
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsGoConfig "github.com/aws/aws-sdk-go-v2/config" // Alias para evitar conflito com pkg/config
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager" // Para S3 Upload Manager
	"github.com/aws/aws-sdk-go-v2/service/s3/types" // Para types.NoSuchKey
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
		phxlog.L.Warn("AWS_S3_BUCKET not set. File upload to S3 will be disabled.")
		return nil, nil
	}
	if region == "" {
		phxlog.L.Warn("AWS_REGION (for S3) not set. File upload to S3 will be disabled.")
		return nil, nil
	}

	// Carregar configuração AWS SDK (usa credenciais do ambiente: variáveis ou IAM role)
	sdkConfig, err := awsGoConfig.LoadDefaultConfig(context.TODO(), awsGoConfig.WithRegion(region))
	if err != nil {
		phxlog.L.Error("Failed to load AWS SDK config for S3. Ensure AWS credentials and region are configured.", zap.Error(err))
		return nil, fmt.Errorf("failed to load AWS SDK config for S3: %w", err)
	}
	phxlog.L.Info("AWS SDK config loaded successfully for S3", zap.String("region", region))

	s3Client := s3.NewFromConfig(sdkConfig)
	uploader := manager.NewUploader(s3Client)

	phxlog.L.Info("Amazon S3 storage provider initialized", zap.String("bucket", bucket), zap.String("region", region))

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
	// publicURL := uploadOutput.Location // Não retornamos mais a URL diretamente.
	// if publicURL == "" {
	//	publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, objectName)
	// }

	phxlog.L.Info("File uploaded successfully to S3",
		zap.String("bucket", s.bucketName),
		zap.String("objectName", objectName),
		zap.String("location", uploadOutput.Location)) // Adicionar location que é a URL completa
	return objectName, nil // Retorna o objectName (key)
}

// DeleteFile remove um arquivo do S3 usando o objectName (key).
func (s *S3StorageProvider) DeleteFile(ctx context.Context, objectName string) error {
	if s.client == nil || s.bucketName == "" {
		return fmt.Errorf("S3 provider not initialized or configured correctly")
	}
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty for DeleteFile")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectName),
	})

	if err != nil {
		// Em aws-sdk-go-v2, para verificar o tipo de erro específico:
		// if errors.As(err, &nsk)
		// Por simplicidade e compatibilidade com a verificação de string que pode já existir:
		if strings.Contains(err.Error(), "NoSuchKey") { // O SDK v2 pode não retornar NoSuchKey diretamente assim, pode ser um *types.NoSuchKey
			phxlog.L.Info("S3 DeleteFile: Object not found (considered successful for idempotency)",
				zap.String("objectName", objectName),
				zap.String("bucket", s.bucketName))
			return nil
		}
		phxlog.L.Error("Failed to delete object from S3",
			zap.String("objectName", objectName),
			zap.String("bucket", s.bucketName),
			zap.Error(err))
		return fmt.Errorf("failed to delete object '%s' from S3 bucket '%s': %w", objectName, s.bucketName, err)
	}

	phxlog.L.Info("File deleted successfully from S3",
		zap.String("bucket", s.bucketName),
		zap.String("objectName", objectName))
	return nil
}

// GetSignedURL gera uma URL assinada para um objeto no S3.
func (s *S3StorageProvider) GetSignedURL(ctx context.Context, objectName string, durationMinutes int) (string, error) {
	if s.client == nil || s.bucketName == "" {
		return "", fmt.Errorf("S3 provider not initialized or configured correctly")
	}
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty for GetSignedURL")
	}

	presignClient := s3.NewPresignClient(s.client)
	presignedURL, err := presignClient.PresignGetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(objectName),
		},
		s3.WithPresignExpires(time.Duration(durationMinutes)*time.Minute),
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL for S3 object '%s': %w", objectName, err)
	}

	return presignedURL.URL, nil
}


// Ensure newline at end of file
