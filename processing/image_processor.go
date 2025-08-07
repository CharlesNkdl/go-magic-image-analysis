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

	// on va déja corriger les reflets de lumière
	grayForInpaint := gocv.NewMat()
	defer grayForInpaint.Close()
	gocv.CvtColor(sourceMat, &grayForInpaint, gocv.ColorBGRToGray)
	reflectionMask := gocv.NewMat()
	defer reflectionMask.Close()
	gocv.Threshold(grayForInpaint, &reflectionMask, 240, 255, gocv.ThresholdBinary)
	dilateKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer dilateKernel.Close()
	gocv.Dilate(reflectionMask, &reflectionMask, dilateKernel)
	// debug
	gocv.IMWrite("debug_reflection_mask.png", reflectionMask)
	inpaintedMat := gocv.NewMat()
	defer inpaintedMat.Close()
	// Le rayon de 5 pixels définit la zone voisine à utiliser pour la reconstruction.
	gocv.Inpaint(sourceMat, reflectionMask, &inpaintedMat, 5, gocv.Telea)
	// On mets en gris pour simplifier le ttt
	// On pourrait voir pour gérer les couleurs pour aider l'algo a trouver les cartes
	// par couleur mais pour l'instant tranquille
	/*
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
		gocv.Canny(blur, &canny, 100, 200)

		// debug
		gocv.IMWrite("debug_canny_output.png", canny)
	*/ // Au final y a pas mal d'erreur par ce que le bords des cartes est blanc ou noir
	// donc on va faire un prétraitement par canaux de couleur
	channels := gocv.Split(inpaintedMat)
	// On aura channels[0] = Bleu, channels[1] = Vert, channels[2] = Rouge
	defer func() {
		for _, c := range channels {
			c.Close()
		}
	}()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	// Dilater chaque canal individuellement
	dilatedB := gocv.NewMat()
	dilatedG := gocv.NewMat()
	dilatedR := gocv.NewMat()
	defer dilatedB.Close()
	defer dilatedG.Close()
	defer dilatedR.Close()

	gocv.Dilate(channels[0], &dilatedB, kernel)
	gocv.Dilate(channels[1], &dilatedG, kernel)
	gocv.Dilate(channels[2], &dilatedR, kernel)

	maxIntensity := gocv.NewMat()
	defer maxIntensity.Close()
	gocv.Max(dilatedB, dilatedG, &maxIntensity)
	gocv.Max(maxIntensity, dilatedR, &maxIntensity)
	// debug
	gocv.IMWrite("debug_max_intensity.png", maxIntensity)
	minIntensity := gocv.NewMat()
	defer minIntensity.Close()
	gocv.Min(channels[0], channels[1], &minIntensity)
	gocv.Min(minIntensity, channels[2], &minIntensity)
	// debug
	gocv.IMWrite("debug_min_intensity.png", minIntensity)
	gradient := gocv.NewMat()
	defer gradient.Close()
	gocv.Subtract(maxIntensity, minIntensity, &gradient)
	//debug
	gocv.IMWrite("debug_gradient.png", gradient)
	binaryEdges := gocv.NewMat()
	defer binaryEdges.Close()
	gocv.Threshold(gradient, &binaryEdges, 20, 255, gocv.ThresholdBinary)
	//debug
	gocv.IMWrite("debug_binary_edges.png", binaryEdges)

	// on repare les contours après l'inpainting
	closeKernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 9, Y: 9})
	defer closeKernel.Close()
	closedEdges := gocv.NewMat()
	defer closedEdges.Close()
	gocv.MorphologyEx(binaryEdges, &closedEdges, gocv.MorphClose, closeKernel)
	gocv.IMWrite("debug_closed_edges.png", closedEdges)

	// après le prettt vraiment sommaire pour l'instant
	// on essaye d'isoler les contours
	// donc pour trouver un truc rectangulaire avec une forme de carte
	contours := gocv.FindContours(closedEdges, gocv.RetrievalExternal, gocv.ChainApproxSimple)
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
		// après quelque test je me retrouve parfois a juste topper le rectangle de texte
		// plutot que la carte en entière
		// on va filtrer le ratio via bounding box parce que la boite de texte est plus large que haute
		boundingRect := gocv.BoundingRect(contour)

		width := float64(boundingRect.Dx())
		height := float64(boundingRect.Dy())
		aspectRatio := 0.0
		// debug
		fmt.Printf("Ratio de la carte détectée: %.2f (largeur: %.2f, hauteur: %.2f)\n", width/height, width, height)
		if width > height {
			aspectRatio = width / height
		} else {
			aspectRatio = height / width
		}
		const minRatio = 1.10
		const maxRatio = 2.20

		if aspectRatio < minRatio || aspectRatio > maxRatio {
			// debug
			fmt.Printf("Contour rejeté : ratio d'aspect de %.2f hors de la plage [%.2f, %.2f]\n", aspectRatio, minRatio, maxRatio)
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
