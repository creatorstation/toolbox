package db

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db_ *gorm.DB

// Connect establishes a connection to the database
func Connect() {
	dsn := os.Getenv("SUPABASE_DSN")

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			Colorful:      false,
		},
	)

	var err error
	db_, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}

func GetDB() *gorm.DB {
	return db_
}
