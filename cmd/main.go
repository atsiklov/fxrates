package main

import (
	"context"
	"fxrates/internal/config"
	"fxrates/internal/config/db"
	"fxrates/internal/config/http"
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)
	appCfg := config.Init()

	ctx := context.Background()
	pool := db.CreatePool(ctx, appCfg.DbServer)
	if pool == nil {
		panic("")
	} // todo: ...
	defer pool.Close()
	log.Println("Successfully connected to DB")

	http.StartServer(appCfg.HTTPServer)
}
