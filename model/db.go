package model

import "gorm.io/gorm"

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	err := db.AutoMigrate(&Section{})
	if err != nil {
		panic(err)
	}
	return &DB{db: db}
}
