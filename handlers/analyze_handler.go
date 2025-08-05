package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CharlesNkdl/go-magic-image-analysis/processing"
	scryfall "github.com/CharlesNkdl/go-scryfall-client/scryfall"

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

	// 5. Appeler votre logique de traitement (que vous placerez dans processing/image_processor.go)
	rawName, err := processing.ExtractCardNameFromMat(mat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur OCR: %v", err)})
		return
	}

	// 6. Nettoyer le texte et appeler Scryfall
	cleanName := strings.TrimSpace(rawName)
	if cleanName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucun nom de carte n'a pu être détecté"})
		return
	}

	cardData, err := scryfall.SearchCardFuzzy(cleanName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":         fmt.Sprintf("Carte non trouvée sur Scryfall pour la recherche: '%s'", cleanName),
			"detected_text": cleanName,
		})
		return
	}

	// 7. Renvoyer les données de la carte en JSON
	c.JSON(http.StatusOK, cardData)
}
