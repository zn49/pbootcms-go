package front

import (
	"crypto/rand"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strconv"

	"github.com/zn49/pbootcms-go/internal/app"
)

// Captcha writes a 4-digit PNG and stores it on the front session.
func Captcha(a *app.App, w http.ResponseWriter, r *http.Request) {
	s := ensureFront(a, w, r)
	var b [2]byte
	_, _ = rand.Read(b[:])
	code := strconv.Itoa(1000 + (int(b[0])<<8|int(b[1]))%9000)
	s.Values["code"] = code
	_ = a.Front.Save(s)
	img := image.NewRGBA(image.Rect(0, 0, 90, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 90; x++ {
			img.Set(x, y, color.RGBA{230, 240, 236, 255})
		}
	}
	for i := 0; i < 4; i++ {
		drawDigit(img, 8+i*20, 6, code[i]-'0')
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_ = png.Encode(w, img)
}

// QR draws a deterministic matrix from ?s=. shortcut: not a standards QR;
// swap in a QR library when scanners must work (network blocked install).
func QR(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("s")
	if s == "" {
		s = r.URL.Query().Get("data")
	}
	if s == "" {
		s = "pbootcms-go"
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	seed := h.Sum64()
	n := 25
	cell := 6
	img := image.NewRGBA(image.Rect(0, 0, n*cell, n*cell))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			on := ((seed >> ((x + y*3) % 61)) & 1) == 1
			if x < 7 && y < 7 || x >= n-7 && y < 7 || x < 7 && y >= n-7 {
				on = finderOn(x%7, y%7) || (x >= n-7 && finderOn(x-(n-7), y)) || (y >= n-7 && finderOn(x, y-(n-7)))
			}
			c := color.White
			if on {
				c = color.Black
			}
			for dy := 0; dy < cell; dy++ {
				for dx := 0; dx < cell; dx++ {
					img.Set(x*cell+dx, y*cell+dy, c)
				}
			}
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_ = png.Encode(w, img)
}

func finderOn(x, y int) bool {
	if x == 0 || y == 0 || x == 6 || y == 6 {
		return true
	}
	return x >= 2 && x <= 4 && y >= 2 && y <= 4
}

var digits = [10][7]string{
	{"111", "101", "101", "101", "101", "101", "111"},
	{"010", "110", "010", "010", "010", "010", "111"},
	{"111", "001", "001", "111", "100", "100", "111"},
	{"111", "001", "001", "111", "001", "001", "111"},
	{"101", "101", "101", "111", "001", "001", "001"},
	{"111", "100", "100", "111", "001", "001", "111"},
	{"111", "100", "100", "111", "101", "101", "111"},
	{"111", "001", "001", "001", "001", "001", "001"},
	{"111", "101", "101", "111", "101", "101", "111"},
	{"111", "101", "101", "111", "001", "001", "111"},
}

func drawDigit(img *image.RGBA, x, y int, d byte) {
	if d > 9 {
		return
	}
	for row, line := range digits[d] {
		for col := 0; col < len(line); col++ {
			if line[col] != '1' {
				continue
			}
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					img.Set(x+col*4+dx, y+row*3+dy, color.RGBA{20, 80, 60, 255})
				}
			}
		}
	}
}
