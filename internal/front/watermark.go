package front

import (
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Watermark draws text onto a jpg/png file in place.
func Watermark(path, text string) {
	if text == "" {
		text = "pbootcms-go"
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	img, format, err := image.Decode(f)
	f.Close()
	if err != nil {
		return
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	col := image.NewUniform(image.Black)
	(&font.Drawer{Dst: dst, Src: col, Face: basicfont.Face7x13, Dot: fixed.P(8, b.Dy()-8)}).DrawString(text)
	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()
	if format == "png" {
		_ = png.Encode(out, dst)
	} else {
		_ = jpeg.Encode(out, dst, &jpeg.Options{Quality: 85})
	}
}
