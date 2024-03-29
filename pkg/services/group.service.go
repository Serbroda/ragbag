package services

import (
	"errors"
	"fmt"

	"github.com/Serbroda/ragbag/pkg/database"
	"github.com/Serbroda/ragbag/pkg/models"
)

var ErrEntityNotFound = errors.New("can not find entity")

func FindGroupById(id string) (models.Group, error) {
	var entity models.Group
	result := database.GetConnection().Where("id = ?", id).Find(&entity)

	if result.RowsAffected == 0 {
		return models.Group{}, ErrEntityNotFound
	}
	return entity, nil
}

func FindGroupsByOwnerId(ownerId uint) []models.Group {
	var entities []models.Group
	database.GetConnection().Where("owner_id = ?", ownerId).Find(&entities)
	return entities
}

func FindLatestGroups(ownerId uint, order string, limit int) []models.Group {
	var entities []models.Group
	sql := `select groups.* 
		from groups 
			left join (
				select distinct group_id, max(updated_at) as updated_at 
				from links l 
				group by group_id
			) links on links.group_id = groups.id 
		where groups.deleted_at is null and groups.owner_id = ?`

	if order != "" {
		sql += " ORDER BY " + order + " "
	}
	if limit > 0 {
		sql += fmt.Sprint(" LIMIT ", limit)
	}
	database.GetConnection().Raw(sql, ownerId).Scan(&entities)
	return entities
}

func FindGroupSubscriptions(userId uint) []models.GroupSubscription {
	var entities []models.GroupSubscription
	database.GetConnection().Preload("Group").Where("user_id = ?", userId).Find(&entities)
	return entities
}

func FindGroupSubscription(userId uint, groupId string) (models.GroupSubscription, error) {
	var entity models.GroupSubscription
	result := database.GetConnection().Where("user_id = ? and group_id = ?", userId, groupId).Find(&entity)
	if result.RowsAffected == 0 {
		return models.GroupSubscription{}, ErrEntityNotFound
	}
	return entity, nil
}
