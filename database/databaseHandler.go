package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"stayinthelan.com/alarm/authentication"

	_ "modernc.org/sqlite"
)

/*
Creates the table for future usage, name: Passwords

	    returns:
		db: the database if everything went right
		error: the error if something went wrong along the way
*/
func CreateTable() (db *sql.DB, error error) {

	db, err := sql.Open("sqlite", "/mnt/persistence/user_data.db")
	if err != nil {
		zap.L().Error("Error when opening Database", zap.Error(err), zap.String("method", "CreateTable"))
		return nil, err

	}
	zap.L().Info("Database connected succesfully")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Passwords(name TEXT PRIMARY KEY, password TEXT, valid_till TEXT);")
	if err != nil {
		zap.L().Error("Error when creating table", zap.Error(err), zap.String("method", "CreateTable"))
		return nil, err
	}
	zap.L().Info("Table created successfully")
	return db, nil

}

/*
Adds or updates a record, with name, and TTL

	params:
	db: Reference to database
	name: username
	ttl: The record's time to live in hours

	returns:
	true when successful, false if not
*/
func AddRecord(db *sql.DB, name string, ttl uint64) bool {
	// For the possibility to update a record
	RemoveRecord(db, name)
	// Create input record and Insert
	password := uuid.New().String()
	authentication.CreateQRCode(fmt.Sprintf("BASEURL/%s?password=%s", name, password), fmt.Sprintf("/mnt/persistence/%s.png", name))
	password = authentication.HashPassword(password)
	validTill := time.Now().UTC().Add(time.Duration(ttl) * time.Hour).Format("2006-01-02 15:04:05")

	_, err := db.Exec("INSERT INTO Passwords(name, password, valid_till) VALUES(?,?,?)", name, password, validTill)
	if err != nil {
		zap.L().Error("Error when inserting record", zap.Error(err))
		return false
	}
	return true
}

/*
Removes the user's record, given by
params:

	db: Database reference
	name: username,

returns:

	true, when successful or no matching record found, false when unsuccessful
*/
func RemoveRecord(db *sql.DB, name string) bool {
	_, err := db.Exec("DELETE FROM Passwords WHERE name == ?", name)
	if err == sql.ErrNoRows {
		zap.L().Info(fmt.Sprintf("No record found for user %s", name), zap.String("method", "RemoveRecord"))
		return true
	} else if err != nil {
		zap.L().Error("Error when removing record", zap.Error(err))
		return false
	}
	err = os.Remove(fmt.Sprintf("/mnt/persistence/%s.png", name))
	if err != nil {
		zap.L().Error("Error occured when removing file", zap.Error(err), zap.String("method", "RemoveInvalidRecords"))
	}

	return true
}

/*
Function for goroutine to periodically clean the database from invalid records and qr-codes
*/
func RemoveInvalidRecords(db *sql.DB, ctx context.Context) {
	// called from API shouldn't tick every hour
	if ctx == nil {
		removeInvalidRecords(db)
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			removeInvalidRecords(db)
		case <-ctx.Done():
			return
		}

	}
}

func removeInvalidRecords(db *sql.DB) {
	rows, err := db.Query("SELECT name FROM Passwords WHERE julianday(valid_till) < julianday('now')")
	if err != nil {
		zap.L().Error("Error occured when querying database", zap.Error(err), zap.String("method", "RemoveInvalidRecords"))
	}
	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			zap.L().Error("Error occured when iterating rows", zap.Error(err), zap.String("method", "RemoveInvalidRecords"))
			continue
		}
		err = os.Remove(fmt.Sprintf("/mnt/persistence/%s.png", name))
		if err != nil {
			zap.L().Error("Error occured when removing file", zap.Error(err), zap.String("method", "RemoveInvalidRecords"))
		}
	}

	_, err = db.Exec("DELETE FROM Passwords WHERE julianday(valid_till) < julianday('now')")

	if err != nil {
		zap.L().Error("Error when removing invalid records", zap.Error(err))
	}
	zap.L().Info("Invalid records removed")
}

// DEBUG
func printOutDatabase(db *sql.DB) {
	rows, err := db.Query("SELECT * FROM Passwords")
	if err != nil {
		zap.L().Error("Error occured when querying database", zap.Error(err), zap.String("method", "PrintOutDatabase"))
	}
	var name, password, valid_till string
	fmt.Printf("---- PRINTING OUT DATABASE ----\n")
	for rows.Next() {
		if err := rows.Scan(&name, &password, &valid_till); err != nil {
			zap.L().Error("Error occured when iterating rows", zap.Error(err), zap.String("method", "PrintOutDatabase"))
			continue
		}
		fmt.Printf("%s %s, %s\n", name, password, valid_till)

	}
}
