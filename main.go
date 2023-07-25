package main

import (
	"github.com/webmakom-com/saiStorage/config"
	"github.com/webmakom-com/saiStorage/server"
)

func main() {
	cfg := config.Load()
	srv := server.NewServer(cfg, false)

	srv.Start()
}
