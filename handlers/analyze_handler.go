package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
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

	extractedCards, err := processing.FindAndExtractCards(mat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erreur lors de la détection de cartes: %v", err)})
		return
	}

	if len(extractedCards) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aucune carte n'a pu être détectée dans l'image"})
		return
	}

	// Traiter chaque carte détectée en parallèle pour plus de rapidité
	var wg sync.WaitGroup
	resultsChan := make(chan gin.H, len(extractedCards))
	client := scryfall.NewClient()
	cardService := client.Cards

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := range extractedCards {
		wg.Add(1)
		go func(cardMat gocv.Mat) {
			defer wg.Done()
			defer cardMat.Close() // Important de fermer chaque Mat extraite

			rawName, err := processing.ExtractCardNameFromMat(cardMat)
			if err != nil {
				resultsChan <- gin.H{"status": "error", "error": fmt.Sprintf("Erreur OCR: %v", err)}
				return
			}

			cleanName := strings.TrimSpace(rawName)
			if cleanName == "" {
				resultsChan <- gin.H{"status": "error", "error": "Aucun nom de carte n'a pu être détecté sur cette carte"}
				return
			}

			cardData, err := cardService.GetByName(ctx, &cards.NamedCardParams{
				Fuzzy: &cleanName,
			})
			if err != nil {
				resultsChan <- gin.H{
					"status":        "error",
					"error":         fmt.Sprintf("Carte non trouvée sur Scryfall pour la recherche: '%s'", cleanName),
					"detected_text": cleanName,
				}
				return
			}
			resultsChan <- gin.H{"status": "success", "data": cardData}
		}(extractedCards[i])
	}

	wg.Wait()
	close(resultsChan)

	var finalResults []gin.H
	for result := range resultsChan {
		finalResults = append(finalResults, result)
	}

	c.JSON(http.StatusOK, finalResults)
}
