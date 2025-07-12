package models

import (
	"phoenixgrc/backend/internal/utils"

	"gorm.io/gorm"
)

// SystemSetting armazena configurações globais do sistema no banco de dados.
// Isso permite que as configurações sejam alteradas dinamicamente através da UI,
// sem a necessidade de reiniciar a aplicação ou alterar arquivos .env.
type SystemSetting struct {
	gorm.Model
	Key           string `gorm:"type:varchar(100);uniqueIndex;not null"` // A chave da configuração (ex: "SMTP_HOST")
	Value         string `gorm:"type:text;not null"`                     // O valor da configuração, criptografado
	Description   string `gorm:"type:varchar(255)"`                      // Uma breve descrição do que a configuração faz
	IsEncrypted   bool   `gorm:"default:true"`                           // Indica se o valor está criptografado
	ExposedToUI   bool   `gorm:"default:true"`                           // Controla se esta configuração deve ser exposta na UI de admin
	Validation    string `gorm:"type:varchar(100)"`                      // Regra de validação (ex: "email", "url", "not_empty")
}

// BeforeSave é um hook do GORM que criptografa o valor antes de salvar.
func (s *SystemSetting) BeforeSave(tx *gorm.DB) (err error) {
	if s.IsEncrypted && s.Value != "" {
		encryptedValue, err := utils.Encrypt(s.Value)
		if err != nil {
			return err
		}
		s.Value = encryptedValue
	}
	return nil
}

// GetDecryptedValue descriptografa e retorna o valor da configuração.
func (s *SystemSetting) GetDecryptedValue() (string, error) {
	if !s.IsEncrypted || s.Value == "" {
		return s.Value, nil
	}
	return utils.Decrypt(s.Value)
}

// GetSystemSetting busca uma configuração específica no banco de dados e retorna seu valor descriptografado.
func GetSystemSetting(db *gorm.DB, key string) (string, error) {
	var setting SystemSetting
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		return "", err
	}
	return setting.GetDecryptedValue()
}
