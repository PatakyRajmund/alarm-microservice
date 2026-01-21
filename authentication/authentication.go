package authentication

import (
	"database/sql"
	"fmt"
	"os"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"rsc.io/qr"
)

/*
Authenticate user

params:

	db: Database Reference
	name: username
	passwordToCheck: The sent password, usually comes in queryParams through API
*/
func Authenticate(db *sql.DB, name string, passwordToCheck string) bool {
	var hash string
	if passwordToCheck == "" {
		zap.L().Error("Password is empty", zap.String("method", "Authenticate"))
		return false
	}
	err := db.QueryRow("SELECT password FROM Passwords WHERE name = ? AND julianday(valid_till) > julianday('now')", name).Scan(&hash)
	if err == sql.ErrNoRows {
		zap.L().Warn(fmt.Sprintf("No valid records found for %s", name), zap.String("method", "Authenticate"))
		return false
	} else {
		if err != nil {
			zap.L().Error("Error when Authenticating", zap.Error(err))
			return false
		}
	}

	return checkPassword(hash, passwordToCheck)
}

/*
Criptographically hashing the password, with salting for extra safety

params:

	password: The password to hash
*/
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		zap.L().Error("Error when Hashing password", zap.Error(err))
		return ""
	} else {
		return string(bytes)
	}
}

/*
Checking if the hashed password matches the given password when authenticating

params:

	hash: The hashed password from the database
	passwordToCheck: The password that was given by authentication UNHASHED
*/
func checkPassword(hash string, passwordToCheck string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwordToCheck))
	if err != nil {
		zap.L().Warn("Error when checking password", zap.Error(err))
		return false
	}
	return true
}

/*
Creating QRCode with

	params:
	content: The QR code content
	filename: The name of the output file
*/
func CreateQRCode(content string, filename string) {
	code, err := qr.Encode(content, qr.M)
	if err != nil {
		zap.L().Error("Error when encoding qr code's content", zap.Error(err))
		return
	}
	pngBytes := code.PNG()
	err = os.WriteFile(filename, pngBytes, 0644)
	if err != nil {
		zap.L().Error("Error when writing file", zap.Error(err), zap.String("method", "CreateQRCode"))
		return
	}
	zap.L().Info(fmt.Sprintf("QR code created and saved to file %s", filename))
}
