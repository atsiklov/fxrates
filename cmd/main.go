package main

import (
	"fxrates/internal/config"
	"fxrates/internal/config/http"
)

func main() {
	appCfg := config.Init()

	// pool := db.CreatePool(nil, appCfg.DbServer) // todo: add ctx
	// if pool == nil {panic("")} // todo: ...
	// defer pool.Close()

	http.StartServer(appCfg.HTTPServer)
}
