package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GenerateLabelPNG génère une étiquette PNG avec QR et texte et écrit sur writer
func GenerateLabelPNG(dbPath *PartMeta, url string, w io.Writer) error {
	if dbPath == nil || !dbPath.Found {
		return fmt.Errorf("pièce introuvable")
	}

	// Contenu du QR : JSON avec id, type, name et URL
	payload := map[string]interface{}{
		"id":   dbPath.ID,
		"type": dbPath.Type,
		"name": dbPath.Name,
		"url":  url,
	}
	payloadBytes, _ := json.Marshal(payload)

	qr, err := qrcode.New(string(payloadBytes), qrcode.Medium)
	if err != nil {
		return err
	}
	qrImg := qr.Image(256)

	// Dessiner texte + QR sur un canvas plus grand
	textLines := []string{
		fmt.Sprintf("#%d", dbPath.ID),
		dbPath.Name,
		dbPath.Type,
	}
	if dbPath.LocationPath != "" {
		textLines = append(textLines, dbPath.LocationPath)
	}

	// Dimensions
	qrSize := qrImg.Bounds().Dx()
	padding := 10
	lineHeight := 14
	textHeight := lineHeight * len(textLines)
	width := qrSize + padding*2
	height := qrSize + padding*3 + textHeight

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// Placer le QR
	offset := image.Pt(padding, padding)
	draw.Draw(canvas, image.Rect(offset.X, offset.Y, offset.X+qrSize, offset.Y+qrSize), qrImg, image.Point{}, draw.Src)

	// Dessiner texte sous le QR
	face := basicfont.Face7x13
	y := padding + qrSize + padding + face.Ascent
	for _, line := range textLines {
		addLabelText(canvas, line, padding, y, face, black)
		y += lineHeight
	}

	// Encoder en PNG
	return png.Encode(w, canvas)
}

func addLabelText(img *image.RGBA, text string, x, y int, face font.Face, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// DefaultLabelURL construit une URL standardisée pour la pièce
func DefaultLabelURL(id int) string {
	// Utiliser un schéma interne; peut être remplacé par une URL publique
	return fmt.Sprintf("recycle://view/%d", id)
}
