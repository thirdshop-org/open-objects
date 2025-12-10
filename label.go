package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/url"

	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GenerateLabelPNG génère une étiquette PNG avec QR et texte et écrit sur writer
func GenerateLabelPNG(dbPath *PartMeta, qrContent string, w io.Writer) error {
	if dbPath == nil || !dbPath.Found {
		return fmt.Errorf("pièce introuvable")
	}

	// Contenu du QR : préfixe PRT-<id>
	qr, err := qrcode.New(qrContent, qrcode.Medium)
	if err != nil {
		return err
	}
	qrImg := qr.Image(256)

	// Dessiner texte + QR sur un canvas plus grand
	textLines := []string{
		fmt.Sprintf("PRT-%d", dbPath.ID),
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

// DefaultLabelQRContent construit le contenu QR standard pour une pièce
func DefaultLabelQRContent(id int) string {
	return fmt.Sprintf("PRT-%d", id)
}

// DefaultLocationLabelURL construit l'URL encodée pour une localisation (fallback)
func DefaultLocationLabelURL(path string) string {
	escaped := url.QueryEscape(path)
	return fmt.Sprintf("/location?path=%s", escaped)
}

// GenerateLocationLabelPNG génère une étiquette PNG pour un lieu (ID + path) avec QR
func GenerateLocationLabelPNG(locID int, path, qrContent string, w io.Writer) error {
	if locID <= 0 {
		return fmt.Errorf("id localisation invalide")
	}
	if path == "" {
		return fmt.Errorf("path vide")
	}

	qr, err := qrcode.New(qrContent, qrcode.Medium)
	if err != nil {
		return err
	}
	qrImg := qr.Image(256)

	textLines := []string{
		fmt.Sprintf("LOC-%d", locID),
		path,
	}

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

	offset := image.Pt(padding, padding)
	draw.Draw(canvas, image.Rect(offset.X, offset.Y, offset.X+qrSize, offset.Y+qrSize), qrImg, image.Point{}, draw.Src)

	face := basicfont.Face7x13
	y := padding + qrSize + padding + face.Ascent
	for _, line := range textLines {
		addLabelText(canvas, line, padding, y, face, black)
		y += lineHeight
	}

	return png.Encode(w, canvas)
}
