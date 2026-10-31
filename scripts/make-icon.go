// Command make-icon generates placeholder app art from stdlib only:
//
//	go run scripts/make-icon.go
//
// Outputs build/appicon.png (Wails packager source) and
// build/windows/icon.ico (PNG-in-ICO, Vista+; used by the exe + NSIS).
// Replace with real artwork any time — filenames stay the same.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const size = 256

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// Transparent background.
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)
	// Dark rounded square.
	bg := color.RGBA{R: 0x12, G: 0x18, B: 0x1F, A: 0xFF}
	radius := 56.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := math.Min(float64(x), float64(size-1-x))
			dy := math.Min(float64(y), float64(size-1-y))
			in := true
			if dx < radius && dy < radius {
				cx, cy := radius-dx, radius-dy
				in = cx*cx+cy*cy <= radius*radius
			}
			if in {
				img.Set(x, y, bg)
			}
		}
	}
	// Blue disc.
	cx, cy, rad := 128.0, 128.0, 84.0
	blue := color.RGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy <= rad*rad {
				img.Set(x, y, blue)
			}
		}
	}
	// White paper-plane triangle pointing up-right.
	plane := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	pts := [][2]float64{{88, 168}, {168, 88}, {140, 176}, {120, 140}, {84, 152}}
	_ = pts
	tri := [3][2]float64{{92, 164}, {164, 92}, {112, 150}}
	for y := 60; y < 200; y++ {
		for x := 60; x < 200; x++ {
			if pointInTri(float64(x), float64(y), tri) {
				img.Set(x, y, plane)
			}
		}
	}
	// Fold line (darker blue) for depth.
	fold := color.RGBA{R: 0x1D, G: 0x4E, B: 0xD8, A: 0xFF}
	for i := 0; i <= 60; i++ {
		t := float64(i) / 60
		img.Set(int(164-52*t), int(92+58*t), fold)
		img.Set(int(164-52*t)+1, int(92+58*t), fold)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll("build/windows", 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join("build", "appicon.png"), pngBuf.Bytes(), 0o644); err != nil {
		fatal(err)
	}
	ico := buildICO(pngBuf.Bytes())
	if err := os.WriteFile(filepath.Join("build", "windows", "icon.ico"), ico, 0o644); err != nil {
		fatal(err)
	}
	fmt.Println("wrote build/appicon.png + build/windows/icon.ico")
}

func pointInTri(x, y float64, t [3][2]float64) bool {
	sign := func(ax, ay, bx, by, cx, cy float64) float64 {
		return (ax-cx)*(by-cy) - (bx-cx)*(ay-cy)
	}
	d1 := sign(x, y, t[0][0], t[0][1], t[1][0], t[1][1])
	d2 := sign(x, y, t[1][0], t[1][1], t[2][0], t[2][1])
	d3 := sign(x, y, t[2][0], t[2][1], t[0][0], t[0][1])
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// buildICO wraps one PNG blob as a single-image Vista+ ICO.
func buildICO(pngBlob []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type: icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // count
	out.Write([]byte{0, 0, 0, 0}) // w,h,colors,reserved (0,0 = 256)
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // planes
	_ = binary.Write(&out, binary.LittleEndian, uint16(32))
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(pngBlob)))
	_ = binary.Write(&out, binary.LittleEndian, uint32(6+16))
	out.Write(pngBlob)
	return out.Bytes()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "make-icon:", err)
	os.Exit(1)
}
