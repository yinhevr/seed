package model

import (
	"github.com/go-xorm/xorm"

	"time"
)

var database *xorm.Engine

// InitDB ...
func InitDB() (e error) {
	db, e := xorm.NewEngine("sqlite3", "seed.db")
	if e != nil {
		return e
	}
	db.ShowSQL(true)
	db.ShowExecTime(true)
	e = db.Sync2(&Video{})
	if e != nil {
		return e
	}
	database = db
	return nil
}

// Model ...
type Model struct {
	ID        string     `json:"-" xorm:"id"`
	CreatedAt time.Time  `json:"-" xorm:"created_at"`
	UpdatedAt time.Time  `json:"-" xorm:"updated_at"`
	DeletedAt *time.Time `json:"-" xorm:"deleted_at"`
	Version   int        `json:"-" xorm:"version"`
}
