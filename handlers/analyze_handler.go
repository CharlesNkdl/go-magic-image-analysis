package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/CharlesNkdl/go-magic-image-analysis/processing"
	scryfall "github.com/CharlesNkdl/go-scryfall-client/scryfall"
	"github.com/CharlesNkdl/go-scryfall-client/scryfall/models/request/cards"

	"github.com/gin-gonic/gin"
	"gocv.io/x/gocv"
)

func AnalyzeCardHandler(c *gin.Context) {
	file, err := c.FormFile("cardImage")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Aucun fichier 'cardImage' n'a été fourni"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'ouvrir le fichier"})
		return
	}
	defer src.Close()

	imageBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de lire le fichier"})
		return
	}

	mat, err := gocv.IMDecode(imageBytes, gocv.IMReadColor)
	if err != nil || mat.Empty() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de décoder l'image"})
		return
	}
	defer mat.Close()

	rawName, err := processing.ExtractCardNameFromMat(mat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur OCR: %v", err)})
		return
	}

	cleanName := strings.TrimSpace(rawName)
	if cleanName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucun nom de carte n'a pu être détecté"})
		return
	}
	client := scryfall.NewClient()
	cardService := client.Cards

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cardData, err := cardService.GetByName(ctx, &cards.NamedCardParams{
		Fuzzy: &cleanName,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         fmt.Sprintf("Carte non trouvée sur Scryfall pour la recherche: '%s'", cleanName),
			"detected_text": cleanName,
		})
		return
	}
	c.JSON(http.StatusOK, cardData)
}
