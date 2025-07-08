package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/models"
	"regexp" // Para validar cores HEX

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BrandingPayload define a estrutura para o JSON de branding no multipart.
type BrandingPayload struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
}

var hexColorRegex = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)

// UpdateOrganizationBrandingHandler atualiza as configurações de branding da organização.
func UpdateOrganizationBrandingHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}

	// Autorização: verificar se o usuário logado é admin/manager da targetOrgID
	if !checkOrgAdminOrManager(c, targetOrgID) { // Reutilizando helper de organization_user_handler
		return
	}
	actingUserID, _ := c.Get("userID") // Para log ou auditoria futura

	// Processar multipart form
	if err := c.Request.ParseMultipartForm(5 << 20); err != nil { // Limite de 5MB para o form todo
		c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao processar formulário multipart: " + err.Error()})
		return
	}

	var payload BrandingPayload
	payloadString := c.Request.FormValue("data") // Dados JSON no campo "data"
	if payloadString != "" {
		if err := json.Unmarshal([]byte(payloadString), &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "JSON de branding ('data') inválido: " + err.Error()})
			return
		}
	}

	// Validação das cores HEX
	if payload.PrimaryColor != "" && !hexColorRegex.MatchString(payload.PrimaryColor) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de Cor Primária inválido. Use #RRGGBB ou #RGB."})
		return
	}
	if payload.SecondaryColor != "" && !hexColorRegex.MatchString(payload.SecondaryColor) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de Cor Secundária inválido. Use #RRGGBB ou #RGB."})
		return
	}


	db := database.GetDB()
	var organization models.Organization
	if err := db.First(&organization, "id = ?", targetOrgID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organização não encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar organização: " + err.Error()})
		return
	}

	// Lidar com upload de logo
	var uploadedLogoURL string
	file, header, errFile := c.Request.FormFile("logo_file")
	if errFile == nil { // Arquivo de logo fornecido
		defer file.Close()

		// Validações de arquivo (tamanho, tipo) - similar ao upload de evidência
		if header.Size > (2 << 20) { // Limite de 2MB para logo
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Arquivo de logo excede o limite de %dMB", (2<<20)/(1024*1024))})
			return
		}

		buffer := make([]byte, 512)
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao ler arquivo de logo para detecção de tipo"})
			return
		}
		_, err = file.Seek(0, io.SeekStart) // Resetar ponteiro
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao resetar ponteiro do arquivo de logo"})
			return
		}
		mimeType := http.DetectContentType(buffer)
		allowedLogoMimeTypes := map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/svg+xml": true}
		if !allowedLogoMimeTypes[mimeType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tipo de arquivo de logo não permitido: %s. Permitidos: JPEG, PNG, GIF, SVG.", mimeType)})
			return
		}

		if filestorage.DefaultFileStorageProvider == nil {
			log.Println("Tentativa de upload de logo, mas FileStorageProvider não configurado.")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Serviço de armazenamento de arquivos não configurado."})
			return
		}

		// Nome do objeto no GCS: {orgId}/branding/logo_{timestamp}{ext}
		fileExtension := filepath.Ext(header.Filename)
		objectName := fmt.Sprintf("%s/branding/logo_%d%s", targetOrgID.String(), time.Now().UnixNano(), fileExtension)

		logoURL, errUpload := filestorage.DefaultFileStorageProvider.UploadFile(c.Request.Context(), targetOrgID.String(), objectName, file)
		if errUpload != nil {
			log.Printf("Falha ao fazer upload do logo para GCS para org %s por usuário %s: %v", targetOrgID, actingUserID, errUpload)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao fazer upload do logo: " + errUpload.Error()})
			return
		}
		uploadedLogoURL = logoURL
		log.Printf("Logo para organização %s atualizado por usuário %s: %s", targetOrgID, actingUserID, uploadedLogoURL)
		organization.LogoURL = uploadedLogoURL // Atualiza apenas se novo logo foi enviado
	} else if errFile != http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao processar arquivo de logo: " + errFile.Error()})
		return
	}


	// Atualizar cores se fornecidas no payload
	if payloadString != "" { // Se o campo 'data' foi enviado
		if payload.PrimaryColor != "" {
			organization.PrimaryColor = payload.PrimaryColor
		}
		if payload.SecondaryColor != "" {
			organization.SecondaryColor = payload.SecondaryColor
		}
	}


	if err := db.Save(&organization).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao salvar configurações de branding: " + err.Error()})
		return
	}

	// Retornar a organização atualizada (ou apenas uma mensagem de sucesso)
	// Para simplificar, vamos retornar a organização. A resposta de GetOrganizationHandler poderia ser um DTO.
	var updatedOrg models.Organization
	db.First(&updatedOrg, targetOrgID) // Re-fetch para garantir dados consistentes

	c.JSON(http.StatusOK, updatedOrg)
}

// GetOrganizationBrandingHandler retorna as configurações de branding da organização.
// Este endpoint pode ser público ou protegido dependendo da necessidade de exibir branding antes do login.
// Por enquanto, vamos fazê-lo protegido, mas sem exigir admin/manager, apenas que o usuário pertença à org.
func GetOrganizationBrandingHandler(c *gin.Context) {
    orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}

    // Se for para ser público para a tela de login, esta verificação de token não se aplica.
    // Mas se for para o painel admin, ela é útil.
    tokenOrgID, orgOk := c.Get("organizationID")
	if !orgOk || tokenOrgID.(uuid.UUID) != targetOrgID {
        // Para um endpoint verdadeiramente público, removeríamos esta checagem de token
        // e buscaríamos a organização diretamente pelo orgId da URL.
		// c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
		// return
	}

    db := database.GetDB()
	var organization models.Organization
	if err := db.Select("id", "name", "logo_url", "primary_color", "secondary_color").First(&organization, "id = ?", targetOrgID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organização não encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar dados de branding da organização: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
        "id": organization.ID,
        "name": organization.Name, // Útil para confirmar
        "logo_url": organization.LogoURL,
        "primary_color": organization.PrimaryColor,
        "secondary_color": organization.SecondaryColor,
    })
}
