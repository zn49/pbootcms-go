// Generates transparent PbootCMS-Go logos (icon + full wordmark).
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var blue = color.NRGBA{R: 0x4A, G: 0xA8, B: 0xE0, A: 0xFF}
var white = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}

func main() {
	root := `E:\projects\vibe\gbootcms\site`
	icon := drawIcon(256)
	full := drawFull()

	outs := map[string]image.Image{
		filepath.Join(root, "apps", "admin", "view", "default", "images", "logo-icon.png"): icon,
		filepath.Join(root, "static", "images", "logo-icon.png"):                             icon,
		filepath.Join(root, "apps", "admin", "view", "default", "images", "logo.png"):        full,
		filepath.Join(root, "static", "images", "logo.png"):                                  full,
		`C:\Users\user\.cursor\projects\e-projects-vibe-gbootcms\assets\gbootcms-logo.png`:  full,
	}
	for p, img := range outs {
		if err := writePNG(p, img); err != nil {
			panic(err)
		}
		println("wrote", p)
	}
	icoPath := filepath.Join(root, "favicon.ico")
	if err := writeICO(icoPath, []image.Image{scaleNearest(icon, 32), scaleNearest(icon, 16)}); err != nil {
		panic(err)
	}
	println("wrote", icoPath)
}

func scaleNearest(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}

// writeICO embeds PNG frames (Vista+ / all modern browsers).
func writeICO(path string, frames []image.Image) error {
	type entry struct {
		w, h uint8
		png  []byte
	}
	ents := make([]entry, 0, len(frames))
	for _, fr := range frames {
		b := fr.Bounds()
		var buf bytes.Buffer
		if err := png.Encode(&buf, fr); err != nil {
			return err
		}
		w, h := b.Dx(), b.Dy()
		if w > 255 {
			w = 0
		}
		if h > 255 {
			h = 0
		}
		ents = append(ents, entry{uint8(w), uint8(h), buf.Bytes()})
	}
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(ents)))
	offset := 6 + 16*len(ents)
	for _, e := range ents {
		out.WriteByte(e.w)
		out.WriteByte(e.h)
		out.WriteByte(0) // colors
		out.WriteByte(0) // reserved
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))  // planes
		_ = binary.Write(&out, binary.LittleEndian, uint16(32)) // bitcount
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(e.png)))
		_ = binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(e.png)
	}
	for _, e := range ents {
		out.Write(e.png)
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func writePNG(path string, img image.Image) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawIcon(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	// fully transparent canvas
	r := float64(size) * 0.18
	fillRoundRect(img, 0, 0, size-1, size-1, r, blue)
	drawG(img, size)
	return img
}

func drawG(img *image.NRGBA, size int) {
	s := float64(size)
	thick := s * 0.078
	// three speed streaks (like original P wings), left of letter
	for i, yoff := range []float64{-0.11, 0, 0.11} {
		y := s*0.36 + s*yoff
		w := s * (0.20 - float64(i)*0.015)
		strokeLine(img, s*0.12, y, s*0.12+w, y, thick*0.7, white)
	}
	// G as C-arc + mid spur (readable letter, not a gauge)
	cx, cy := s*0.56, s*0.54
	rad := s * 0.26
	// almost full circle with gap on the right (~50°)
	strokeArc(img, cx, cy, rad, thick, 50*math.Pi/180, (360-10)*math.Pi/180, white)
	// spur inward from right opening
	a := 20 * math.Pi / 180
	x1 := cx + math.Cos(a)*rad
	y1 := cy + math.Sin(a)*rad
	strokeLine(img, x1, y1, cx+rad*0.05, cy, thick*0.9, white)
}

func drawFull() *image.NRGBA {
	w, h := 720, 160
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	icon := drawIcon(140)
	blit.DrawOver(img, image.Rect(10, 10, 150, 150), icon, image.Pt(0, 0))

	faceBold := loadFace([]string{
		`C:\Windows\Fonts\arialbd.ttf`,
		`C:\Windows\Fonts\segoeuib.ttf`,
		`C:\Windows\Fonts\msyhbd.ttc`,
	}, 52)
	faceCN := loadFace([]string{
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\msyhbd.ttc`,
	}, 22)
	if faceBold != nil {
		drawString(img, faceBold, "GBOOTCMS", 170, 78, blue)
	}
	if faceCN != nil {
		drawString(img, faceCN, "开源免费Go建站系统", 172, 118, blue)
	}
	return img
}

// tiny blit helper package-local
var blit = struct {
	DrawOver func(dst *image.NRGBA, r image.Rectangle, src image.Image, sp image.Point)
}{
	DrawOver: func(dst *image.NRGBA, r image.Rectangle, src image.Image, sp image.Point) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				sx := sp.X + (x - r.Min.X)
				sy := sp.Y + (y - r.Min.Y)
				sc := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
				if sc.A == 0 {
					continue
				}
				dc := dst.NRGBAAt(x, y)
				a := float64(sc.A) / 255
				dst.SetNRGBA(x, y, color.NRGBA{
					R: uint8(float64(sc.R)*a + float64(dc.R)*(1-a)),
					G: uint8(float64(sc.G)*a + float64(dc.G)*(1-a)),
					B: uint8(float64(sc.B)*a + float64(dc.B)*(1-a)),
					A: uint8(math.Min(255, float64(dc.A)+float64(sc.A))),
				})
			}
		}
	},
}

func loadFace(paths []string, size float64) font.Face {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		ft, err := opentype.Parse(b)
		if err != nil {
			// ttc collection
			c, err2 := opentype.ParseCollection(b)
			if err2 != nil {
				continue
			}
			ft, err = c.Font(0)
			if err != nil {
				continue
			}
		}
		face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if err == nil {
			return face
		}
	}
	return nil
}

func drawString(img *image.NRGBA, face font.Face, s string, x, y int, col color.NRGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func fillRoundRect(img *image.NRGBA, x0, y0, x1, y1 int, rad float64, col color.NRGBA) {
	// soft edge (~1px) so scaled-down icon doesn't look like a dirty cutout
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			d := distOutsideRoundRect(float64(x)+0.5, float64(y)+0.5, float64(x0), float64(y0), float64(x1), float64(y1), rad)
			if d >= 1 {
				continue
			}
			a := col.A
			if d > 0 {
				a = uint8(float64(col.A) * (1 - d))
			}
			img.SetNRGBA(x, y, color.NRGBA{R: col.R, G: col.G, B: col.B, A: a})
		}
	}
}

// signed distance outside rounded rect; 0 = inside/on edge
func distOutsideRoundRect(x, y, x0, y0, x1, y1, r float64) float64 {
	cx := clamp(x, x0+r, x1-r)
	cy := clamp(y, y0+r, y1-r)
	dx, dy := x-cx, y-cy
	if dx == 0 && dy == 0 {
		// inside the flat region
		if x >= x0 && x <= x1 && y >= y0 && y <= y1 {
			return 0
		}
	}
	// distance from nearest point on inner rect
	if x >= x0+r && x <= x1-r && y >= y0 && y <= y1 {
		return 0
	}
	if y >= y0+r && y <= y1-r && x >= x0 && x <= x1 {
		return 0
	}
	d := math.Hypot(dx, dy) - r
	if d < 0 {
		return 0
	}
	return d
}

func clamp(v, a, b float64) float64 {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}

func strokeLine(img *image.NRGBA, x0, y0, x1, y1, thick float64, col color.NRGBA) {
	steps := int(math.Hypot(x1-x0, y1-y0)) + 1
	r := thick / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		fillCircle(img, x, y, r, col)
	}
}

func strokeArc(img *image.NRGBA, cx, cy, rad, thick, a0, a1 float64, col color.NRGBA) {
	steps := int((a1 - a0) * rad * 2)
	if steps < 32 {
		steps = 32
	}
	for i := 0; i <= steps; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(steps)
		x := cx + math.Cos(a)*rad
		y := cy + math.Sin(a)*rad
		fillCircle(img, x, y, thick/2, col)
	}
}

func fillCircle(img *image.NRGBA, cx, cy, r float64, col color.NRGBA) {
	b := img.Bounds()
	x0 := int(cx - r - 1)
	x1 := int(cx + r + 1)
	y0 := int(cy - r - 1)
	y1 := int(cy + r + 1)
	rr := r * r
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
				continue
			}
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= rr {
				// alpha blend over existing
				sc := col
				dc := img.NRGBAAt(x, y)
				if dc.A == 0 {
					img.SetNRGBA(x, y, sc)
					continue
				}
				a := float64(sc.A) / 255
				img.SetNRGBA(x, y, color.NRGBA{
					R: uint8(float64(sc.R)*a + float64(dc.R)*(1-a)),
					G: uint8(float64(sc.G)*a + float64(dc.G)*(1-a)),
					B: uint8(float64(sc.B)*a + float64(dc.B)*(1-a)),
					A: 255,
				})
			}
		}
	}
}
