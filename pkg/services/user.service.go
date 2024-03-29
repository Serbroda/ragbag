package services

import (
	"github.com/Serbroda/ragbag/pkg/database"
	"github.com/Serbroda/ragbag/pkg/models"
)

func FindUserById(id uint) (models.User, error) {
	var entity models.User
	result := database.GetConnection().Where("id = ?", id).Find(&entity)

	if result.RowsAffected == 0 {
		return models.User{}, ErrEntityNotFound
	}
	return entity, nil
}
