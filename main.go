package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"stayinthelan.com/alarm/api"
	"stayinthelan.com/alarm/database"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, _ := zap.NewProduction()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	db, err := database.CreateTable()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &api.ApiHandler{
		DB: db,
	}

	go database.RemoveInvalidRecords(db, ctx)

	router := api.CreateRouter(app)

	log.Fatal(http.ListenAndServe(":8080", router))

}
