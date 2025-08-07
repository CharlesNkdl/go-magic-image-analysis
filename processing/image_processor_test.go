package processing

import (
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
