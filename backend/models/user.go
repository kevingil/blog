package models

type User struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"type:varchar(255);not null"`
	Email        string `json:"email" gorm:"type:varchar(255);not null;unique"`
	PasswordHash string `json:"-" gorm:"column:passwordHash;type:varchar(255);not null"`
	Role         string `json:"role" gorm:"type:varchar(50);not null;default:'user'"`
}

func (User) TableName() string {
	return "users"
}

func (User) SkipSoftDelete() bool {
	return true
}
