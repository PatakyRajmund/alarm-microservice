package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"stayinthelan.com/alarm/authentication"
	"stayinthelan.com/alarm/database"
)

const HOMEASSISTANT_URL = "HA_URL"

type ApiHandler struct {
	DB       *sql.DB
	loggedIn []string
}

func CreateRouter(apiHandler *ApiHandler) *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/api/adduser/{user}/{ttl}", apiHandler.addUserHandler).Methods("POST")
	router.HandleFunc("/api/delete/{user}", apiHandler.deleteUserHandler).Methods("DELETE")
	router.HandleFunc("/api/authenticate/{user}", apiHandler.authenticationHandler).Methods("GET")
	router.HandleFunc("/api/remove-invalid-records", apiHandler.removeInvalidRecordsHandler).Methods("POST")
	router.HandleFunc("/api/getcode/{user}", apiHandler.imageHandler).Methods("GET")
	zap.L().Info("Router created", zap.String("method", "CreateRouter"))
	return router
}

func (h *ApiHandler) imageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]

	http.ServeFile(w, r, fmt.Sprintf("/mnt/persistence/%s.png", username))
}

func (h *ApiHandler) addUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]
	ttlString := vars["ttl"]
	ttl, err := strconv.ParseUint(ttlString, 10, 64)
	if err != nil {
		zap.L().Error("Error occured when parsing ttl string", zap.Error(err), zap.String("method", "ApiHandler.addUserHandler"))
		http.Error(w, "Adding not succesful", 500)
		return
	}

	if database.AddRecord(h.DB, username, ttl) {
		w.Write([]byte("OK"))
		zap.L().Info(fmt.Sprintf("User %s added", username), zap.String("method", "ApiHandler.addUserHandler"))
	} else {
		http.Error(w, "Adding not succesful", 500)
		zap.L().Error(fmt.Sprintf("User %s could not be added", username), zap.String("method", "ApiHandler.addUserHandler"))
	}

}

func (h *ApiHandler) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]
	if database.RemoveRecord(h.DB, username) {
		w.Write([]byte("OK"))
		zap.L().Info(fmt.Sprintf("User %s removed", username), zap.String("method", "ApiHandler.deleteUserHandler"))
	} else {
		http.Error(w, "Could not be removed", 500)
		zap.L().Error(fmt.Sprintf("User %s could not be removed", username), zap.String("method", "ApiHandler.deleteUserHandler"))
	}

}

/*
Handles authentication, AND arms OR disarms alarm based on the # of people at home
(if someone who's already logged in logs in again that counts as a "logout", so they left the house/flat)
*/
func (h *ApiHandler) authenticationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["user"]

	query := r.URL.Query()
	password := query.Get("password")

	if authentication.Authenticate(h.DB, username, password) {
		w.Write([]byte("OK"))
		zap.L().Info(fmt.Sprintf("User %s logged in", username), zap.String("method", "ApiHandler.authenticationHandler"))

		if slices.Contains(h.loggedIn, username) {
			index := slices.Index(h.loggedIn, username)
			lastIndex := len(h.loggedIn) - 1
			if index == lastIndex {
				h.loggedIn = h.loggedIn[:lastIndex]
			} else {
				h.loggedIn[index], h.loggedIn[lastIndex] = h.loggedIn[lastIndex], h.loggedIn[index]
				h.loggedIn = h.loggedIn[:lastIndex]
			}
			// After removing user nobody is logged in => Nobody's home
			if len(h.loggedIn) == 0 {
				_, err := http.Post(fmt.Sprintf("%s/api/webhook/NO_ONE_AT_HOME_WEBHOOK_ID", HOMEASSISTANT_URL), "", nil)
				if err != nil {
					fmt.Println(err)
				}
			}
		} else {
			if len(h.loggedIn) == 0 {
				http.Post(fmt.Sprintf("%s/api/webhook/WEBHOOK_ID", HOMEASSISTANT_URL), "", nil)
			}
			h.loggedIn = append(h.loggedIn, username)
		}

	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		zap.L().Warn(fmt.Sprintf("User %s tried to log in, was unsuccesful", username), zap.String("method", "ApiHandler.authenticationHandler"))
	}
}

func (h *ApiHandler) removeInvalidRecordsHandler(w http.ResponseWriter, r *http.Request) {
	database.RemoveInvalidRecords(h.DB, nil)
}
