package internal

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Page struct {
	BaseEntity   `bson:",inline" json:",inline"`
	Name         string         `bson:"name" json:"name"`
	SpaceID      bson.ObjectID  `bson:"spaceId" json:"spaceId"`
	ParentPageID *bson.ObjectID `bson:"parentPageId,omitempty" json:"parentPageId,omitempty"`
	SubPages     []*Page        `bson:"subPages,omitempty"`
}
