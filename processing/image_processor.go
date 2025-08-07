package processing

import (
	"fmt"
	"image"
	"sort"

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

func FindAndExtractCards(sourceMat gocv.Mat) ([]gocv.Mat, error) {
	if sourceMat.Empty() {
		return nil, fmt.Errorf("la Mat source est vide")
	}
	// On mets en gris pour simplifier le ttt
	// On pourrait voir pour gérer les couleurs pour aider l'algo a trouver les cartes
	// par couleur mais pour l'instant tranquille
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(sourceMat, &gray, gocv.ColorBGRToGray)

	blur := gocv.NewMat()
	defer blur.Close()
	// le blur pour réduire le bruit
	gocv.GaussianBlur(gray, &blur, image.Point{X: 5, Y: 5}, 0, 0, gocv.BorderDefault)

	canny := gocv.NewMat()
	defer canny.Close()
	// on utilise algo de canny pour contours
	// a voir les seuils mais pour l'instant tranquille
	gocv.Canny(blur, &canny, 75, 200)

	// debug
	gocv.IMWrite("debug_canny_output.png", canny)

	// après le prettt vraiment sommaire pour l'instant
	// on essaye d'isoler les contours
	// donc pour trouver un truc rectangulaire avec une forme de carte
	contours := gocv.FindContours(canny, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	var extractedCards []gocv.Mat

	// Étape 3: Filtrer les contours et extraire les cartes
	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		// ca c'est pour filter le bruit, on mets un seuil en dur pour le moment
		// après faudra faire un truc plus intelligent parce que ca depends de la résolution
		area := gocv.ContourArea(contour)
		if area < 5000 {
			continue
		}

		// comme j'ai dit on cherche des rectangles
		approx := gocv.ApproxPolyDP(contour, gocv.ArcLength(contour, true)*0.02, true)
		if approx.Size() != 4 {
			continue
		}

		// la on redresse la carte pour faciliter l'ocr
		points := approx.ToPoints()

		//haut-gauche, haut-droite, bas-droite, bas-gauche
		orderedPoints := orderPoints(points)

		// taille de carte magic grosso modo
		outputWidth := 240
		outputHeight := 336

		sourcePoints := gocv.NewPointVectorFromPoints(orderedPoints)
		destPoints := gocv.NewPointVectorFromPoints([]image.Point{
			{0, 0},
			{outputWidth, 0},
			{outputWidth, outputHeight},
			{0, outputHeight},
		})

		// Calculer la matrice de transformation de perspective
		// merci openCV
		transform := gocv.GetPerspectiveTransform(sourcePoints, destPoints)
		defer transform.Close()

		// Appliquer la transformation
		warpedCard := gocv.NewMat()
		gocv.WarpPerspective(sourceMat, &warpedCard, transform, image.Point{X: outputWidth, Y: outputHeight})

		extractedCards = append(extractedCards, warpedCard)

		sourcePoints.Close()
		destPoints.Close()
	}

	return extractedCards, nil
}

// je prends les 4 pts du contour que je pense avoir detecté
// haut-gauche, haut-droite, bas-droite, bas-gauche
func orderPoints(pts []image.Point) []image.Point {
	// Le point haut-gauche a la plus petite somme (x+y)
	// Le point bas-droite a la plus grande somme (x+y)
	sort.Slice(pts, func(i, j int) bool {
		return (pts[i].X + pts[i].Y) < (pts[j].X + pts[j].Y)
	})

	topLeft := pts[0]
	bottomRight := pts[3]

	// Pour les deux autres, le point haut-droite a la plus petite différence (y-x)
	// et le point bas-gauche a la plus grande différence (y-x)
	remaining := pts[1:3]
	sort.Slice(remaining, func(i, j int) bool {
		return (remaining[i].Y - remaining[i].X) < (remaining[j].Y - remaining[j].X)
	})

	topRight := remaining[0]
	bottomLeft := remaining[1]

	return []image.Point{topLeft, topRight, bottomRight, bottomLeft}
}
