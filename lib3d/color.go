package lib3d

import (
	"strconv"
	"strings"
)

// ====================================================================
// FARBE  (Port von lib-wgl.ts: parseColor + ColorState)
// ====================================================================

// ColorState ist eine RGBA-Farbe mit Werten 0..1.
type ColorState struct {
	R, G, B, A float64
}

// ParseHex parst einen Hex-Farbstring ("#rgb" / "#rrggbb") in einen
// ColorState mit Werten 0..1.
func ParseHex(s string) ColorState {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return ColorState{1, 1, 1, 1}
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return ColorState{1, 1, 1, 1}
	}
	return ColorState{
		R: float64((n>>16)&0xff) / 255,
		G: float64((n>>8)&0xff) / 255,
		B: float64(n&0xff) / 255,
		A: 1,
	}
}

// ParseGray parst eine Graustufe 0..255 in einen ColorState.
func ParseGray(gray float64) ColorState {
	g := gray / 255
	return ColorState{g, g, g, 1}
}

// ParseColorRGB parst r,g,b (0..255) in einen ColorState.
func ParseColorRGB(r, g, b float64) ColorState {
	return ColorState{r / 255, g / 255, b / 255, 1}
}

// ParseColorRGBA parst r,g,b,a (0..255) in einen ColorState.
func ParseColorRGBA(r, g, b, a float64) ColorState {
	return ColorState{r / 255, g / 255, b / 255, a / 255}
}

// ColEq vergleicht zwei Farbzustaende exakt.
func ColEq(a, b ColorState) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

// ====================================================================
// COLORINPUT
//
// Kapselt eine Farbangabe fuer die Style-Objekte. Der Nullwert (Set == false)
// bedeutet "nicht ueberschreiben" im Immediate-Mode. In Go fehlt
// Overloading, daher erzeugen die Helfer (Hex/Gray/RGB/RGBA/ColorOf) einen
// gesetzten ColorInput.
// ====================================================================

// ColorInput ist eine optionale Farbeingabe.
type ColorInput struct {
	Set   bool
	Color ColorState
}

// Hex erzeugt einen gesetzten ColorInput aus einem Hex-Farbstring.
func Hex(s string) ColorInput { return ColorInput{true, ParseHex(s)} }

// Gray erzeugt einen gesetzten ColorInput aus einer Graustufe 0..255.
func Gray(v float64) ColorInput { return ColorInput{true, ParseGray(v)} }

// RGB erzeugt einen gesetzten ColorInput aus r,g,b (0..255).
func RGB(r, g, b float64) ColorInput { return ColorInput{true, ParseColorRGB(r, g, b)} }

// RGBA erzeugt einen gesetzten ColorInput aus r,g,b,a (0..255).
func RGBA(r, g, b, a float64) ColorInput { return ColorInput{true, ParseColorRGBA(r, g, b, a)} }

// ColorOf erzeugt einen gesetzten ColorInput aus einem ColorState.
func ColorOf(c ColorState) ColorInput { return ColorInput{true, c} }
