package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/alexmargomez/docforge-api/internal/application/pool"
	"github.com/alexmargomez/docforge-api/internal/domain/service"
	httpInfra "github.com/alexmargomez/docforge-api/internal/infrastructure/http"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("BACKEND_PORT")
	}
	if port == "" {
		port = "8086"
	}

	workers := 8
	if rawW := os.Getenv("WORKER_POOL_SIZE"); rawW != "" {
		if parsed, err := strconv.Atoi(rawW); err == nil && parsed > 0 {
			workers = parsed
		}
	}

	secret := os.Getenv("DOCFORGE_SIGNING_SECRET")
	signer := service.NewHashSignerService(secret)
	workerPool := pool.NewRenderWorkerPool(workers, signer)
	handler := httpInfra.NewServerHandler(workerPool, signer)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("🚀 DocForge Stream API (Go 1.22) escuchando en http://%s (Workers: %d)", addr, workers)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error fatal en servidor DocForge: %v", err)
	}
}
