package processing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocv.io/x/gocv"
)

func TestExtractCardNameFromMat(t *testing.T) {
	testCases := map[string]string{
		"sol_ring.jpg":       "Sol Ring",
		"lightning_bolt.jpg": "Lightning Bolt",
	}

	imageDir := "../test_images"

	for imageName, expectedName := range testCases {
		t.Run(imageName, func(t *testing.T) {
			imagePath := filepath.Join(imageDir, imageName)
			imgMat := gocv.IMRead(imagePath, gocv.IMReadColor)
			if imgMat.Empty() {
				t.Fatalf("Impossible de lire l'image: %s", imagePath)
			}
			defer imgMat.Close()
			extractedText, err := ExtractCardNameFromMat(imgMat)
			if err != nil {
				t.Fatalf("La fonction a retourné une erreur: %v", err)
			}
			cleanedText := strings.TrimSpace(extractedText)
			if !strings.EqualFold(cleanedText, expectedName) {
				t.Errorf("Nom de carte incorrect. Attendu: '%s', Obtenu: '%s'", expectedName, cleanedText)
			} else {
				t.Logf("Succès ! Attendu: '%s', Obtenu: '%s'", expectedName, cleanedText)
			}
		})
	}
}

func TestFindAndExtractCards(t *testing.T) {
	imagePath := "../test_images/multiple_cards_2.jpg"
	expectedCardCount := 3
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Skipf("Fichier de test non trouvé, test ignoré: %s", imagePath)
		return
	}
	sourceMat := gocv.IMRead(imagePath, gocv.IMReadColor)
	if sourceMat.Empty() {
		t.Fatalf("Impossible de lire l'image de test: %s", imagePath)
	}
	defer sourceMat.Close()
	extractedCards, err := FindAndExtractCards(sourceMat)
	if err != nil {
		t.Fatalf("FindAndExtractCards a retourné une erreur: %v", err)
	}
	defer func() {
		for _, m := range extractedCards {
			m.Close()
		}
	}()
	if len(extractedCards) != expectedCardCount {
		t.Errorf("Nombre de cartes détectées incorrect. Attendu: %d, Obtenu: %d", expectedCardCount, len(extractedCards))
	} else {
		t.Logf("Succès ! %d cartes détectées comme attendu.", expectedCardCount)
	}
	outputDir := "../test_output"
	os.Mkdir(outputDir, 0755)
	for i, cardMat := range extractedCards {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("extracted_card_%d.png", i))
		if ok := gocv.IMWrite(outputPath, cardMat); !ok {
			t.Errorf("Impossible de sauvegarder la carte extraite %d", i)
		} else {
			t.Logf("Carte extraite sauvegardée dans %s", outputPath)
		}
	}
}
