package config

import (
	"github.com/morgine/pkg/config"
	"github.com/morgine/pkg/database/orm"
	"github.com/morgine/pkg/database/postgres"
	"gorm.io/gorm"
)

// 重写 table prefix
func NewPostgresORM(postgresNamespace, gormNamespace, symbol string, configs config.Configs) (*gorm.DB, error) {
	db, err := postgres.NewPostgres(postgresNamespace, configs)
	if err != nil {
		return nil, err
	}
	gormConfig := &orm.Config{}
	err = configs.UnmarshalSub(gormNamespace, gormConfig)
	if err != nil {
		return nil, err
	}
	gormConfig.TablePrefix = symbol + "_"
	return gormConfig.Init(orm.NewPostgresDialector(db))
}
