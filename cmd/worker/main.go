package main

import (
	"log"
	"time"

	"github.com/Lovealone1/nex21-api/internal/platform/config"
)

func main() {
	cfg := config.Load()

	log.Println("🛠 Nex21 Worker started")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Running background jobs for...", cfg.App.Name, "in", cfg.App.Env)
	}
}
