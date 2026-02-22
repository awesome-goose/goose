package sql

import (
	"github.com/awesome-goose/goose/types"
	"gorm.io/gorm"
)

type Db struct {
	*gorm.DB

	log types.Log `inject:""`
}
