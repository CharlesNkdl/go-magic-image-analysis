package processing

import (
	"fmt"
	"image"

	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
)

// ExtractCardNameFromMat prend une image de carte Magic sous forme de Mat GoCV,
// isole la zone du nom, la prétraite et en extrait le texte via OCR.
func ExtractCardNameFromMat(cardMat gocv.Mat) (string, error) {
	if cardMat.Empty() {
		return "", fmt.Errorf("la Mat fournie est vide")
	}
	imgWidth := cardMat.Cols()
	imgHeight := cardMat.Rows()
	roiRect := image.Rect(int(float64(imgWidth)*0.05), int(float64(imgHeight)*0.04), int(float64(imgWidth)*0.95), int(float64(imgHeight)*0.10))

	nameplateMat := cardMat.Region(roiRect)
	defer nameplateMat.Close()

	grayMat := gocv.NewMat()
	defer grayMat.Close()
	gocv.CvtColor(nameplateMat, &grayMat, gocv.ColorBGRToGray)

	binaryMat := gocv.NewMat()
	defer binaryMat.Close()
	gocv.Threshold(grayMat, &binaryMat, 127, 255, gocv.ThresholdBinaryInv)

	//debug
	gocv.IMWrite("debug_nameplate.png", binaryMat)

	buf, err := gocv.IMEncode(gocv.PNGFileExt, binaryMat)
	if err != nil {
		return "", fmt.Errorf("impossible d'encoder l'image pour l'OCR: %v", err)
	}
	defer buf.Close()

	client := gosseract.NewClient()
	defer client.Close()

	client.SetImageFromBytes(buf.GetBytes())
	client.SetLanguage("eng")
	client.SetWhitelist("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("erreur lors de l'extraction de texte par Tesseract: %v", err)
	}

	return text, nil
}
