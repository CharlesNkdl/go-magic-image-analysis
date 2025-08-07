package main

import (
	"github.com/CharlesNkdl/go-magic-image-analysis/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	// Crée un routeur Gin avec les middlewares par défaut (logger, recovery)
	router := gin.Default()
	//router.POST("/")
	// taille max a voir pour modif après
	router.MaxMultipartMemory = 8 << 20
	router.POST("/analyze", handlers.AnalyzeCardHandler)
	router.Run(":8080")
}
