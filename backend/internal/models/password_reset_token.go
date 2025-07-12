package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetToken armazena tokens para a funcionalidade de "esqueci minha senha".
type PasswordResetToken struct {
	gorm.Model
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	User      User      `gorm:"foreignKey:UserID"`
	ExpiresAt time.Time `gorm:"not null"`
}
