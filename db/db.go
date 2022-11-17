package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Cdb struct {
	*gorm.DB
}

var (
	Db *Cdb
)

func InitDb(dsn string) {
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	Db = &Cdb{
		DB: gdb,
	}
}
