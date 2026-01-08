package store

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex"` // Add an index for fast lookups
	PasswordHash string

	// Future-proofing: GORM Association example
	// Posts []Post
}

func (s *Store) CreateUser(username, password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	user := User{
		Username:     username,
		PasswordHash: string(bytes),
	}

	result := s.DB.Create(&user)
	return result.Error
}

func (s *Store) FindUserByUsername(username string) (*User, error) {
	var user User
	result := s.DB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (s *Store) RenameUser(oldName, newName string) error {
	return s.DB.Model(&User{}).
		Where("username = ?", oldName).
		Update("username", newName).Error
}

func (s *Store) RemoveUser(username string) error {
	return s.DB.Unscoped().
		Where("username = ?", username).
		Delete(&User{}).Error
}

func (s *Store) UpdatePassword(username, newPassword string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return err
	}

	return s.DB.Model(&User{}).
		Where("username = ?", username).
		Update("password_hash", string(bytes)).Error
}

func (s *Store) Authenticate(username, password string) (*User, error) {
	var user User

	result := s.DB.Where("username = ?", username).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return &user, nil
}
