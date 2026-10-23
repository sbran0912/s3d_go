// Wegwerf-Generator: baut eine Rakete und schreibt sie als Wavefront-OBJ.
//
// Dieser Code gehoert NICHT zur Bibliothek und soll nach dem einmaligen
// Erzeugen von apps/rocket/rocket.obj wieder entfernt werden (tmp/ loeschen).
//
// Aufruf aus dem Projekt-Root:
//
//	go run ./tmp/gen_rocket
//
// Koordinaten: +Y zeigt nach oben, Basis bei y=0, Spitze bei y=10 (analog
// zur lib3d-Pyramide, damit die Vehicle-Ausrichtung in boids passt).
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type v3 struct{ x, y, z float64 }

type builder struct {
	verts []v3
	tris  [][3]int
	lines [][2]int
}

func (b *builder) addV(p v3) int {
	b.verts = append(b.verts, p)
	return len(b.verts) - 1
}

func (b *builder) tri(a, c, d int) { b.tris = append(b.tris, [3]int{a, c, d}) }
func (b *builder) line(a, c int)   { b.lines = append(b.lines, [2]int{a, c}) }

// buildRocket erzeugt einen geschlossenen Raketenkoerper (Rumpf + Nase +
// Boden) samt 4 Finnen und schreibt zusaetzlich Drahtgitter-Linien.
func buildRocket(b *builder) {
	const segments = 24

	const bodyR = 2.0
	const bodyY0 = 0.0
	const bodyY1 = 6.0
	const tipY = 10.0

	// Zwei Ringe (unten/oben) des Rumpfes.
	ringLo := make([]int, segments)
	ringHi := make([]int, segments)
	for j := 0; j < segments; j++ {
		ang := 2 * math.Pi * float64(j) / segments
		cx, cz := math.Cos(ang), math.Sin(ang)
		ringLo[j] = b.addV(v3{bodyR * cx, bodyY0, bodyR * cz})
		ringHi[j] = b.addV(v3{bodyR * cx, bodyY1, bodyR * cz})
	}
	apex := b.addV(v3{0, tipY, 0})
	baseCenter := b.addV(v3{0, bodyY0, 0})

	for j := 0; j < segments; j++ {
		j1 := (j + 1) % segments

		// Rumpfmantel (aussen, CCW).
		b.tri(ringLo[j], ringHi[j], ringHi[j1])
		b.tri(ringLo[j], ringHi[j1], ringLo[j1])

		// Nasenkegel zum Apex.
		b.tri(ringHi[j], apex, ringHi[j1])

		// Boden-Deckel (zeigt nach -Y).
		b.tri(ringLo[j], ringLo[j1], baseCenter)
	}

	// Drahtgitter: Laengslinien, beide Ringe und Nasenlinien.
	for j := 0; j < segments; j++ {
		j1 := (j + 1) % segments
		b.line(ringLo[j], ringHi[j])
		b.line(ringLo[j], ringLo[j1])
		b.line(ringHi[j], ringHi[j1])
		b.line(ringHi[j], apex)
	}

	// Finnen (4 Stueck): duenne trapezfoermige Platten.
	const finCount = 4
	const finRin = 1.6
	const finRout = 4.5
	const finY0 = 0.0
	const finYout = 2.0
	const finYin = 3.5

	for f := 0; f < finCount; f++ {
		ang := 2 * math.Pi * float64(f) / finCount
		ux, uz := math.Cos(ang), math.Sin(ang)

		a := b.addV(v3{finRin * ux, finY0, finRin * uz})
		c := b.addV(v3{finRout * ux, finY0, finRout * uz})
		d := b.addV(v3{finRout * ux, finYout, finRout * uz})
		e := b.addV(v3{finRin * ux, finYin, finRin * uz})

		b.tri(a, c, d)
		b.tri(a, d, e)

		b.line(a, c)
		b.line(c, d)
		b.line(d, e)
		b.line(e, a)
	}
}

// writeOBJ schreibt das Mesh als Wavefront-OBJ (Vertices, Faces, Lines).
func (b *builder) writeOBJ(path string) error {
	var sb strings.Builder
	sb.WriteString("# rocket.obj - einmalig generiert (Y-up, Basis y=0, Spitze y=10)\n")
	sb.WriteString("# Vertices\n")
	for _, p := range b.verts {
		fmt.Fprintf(&sb, "v %.6f %.6f %.6f\n", p.x, p.y, p.z)
	}
	sb.WriteString("# Faces\n")
	for _, t := range b.tris {
		fmt.Fprintf(&sb, "f %d %d %d\n", t[0]+1, t[1]+1, t[2]+1)
	}
	sb.WriteString("# Lines\n")
	for _, l := range b.lines {
		fmt.Fprintf(&sb, "l %d %d\n", l[0]+1, l[1]+1)
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func main() {
	out := "apps/rocket/rocket.obj"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	b := &builder{}
	buildRocket(b)

	if err := b.writeOBJ(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("geschrieben: %s (%d Vertices, %d Dreiecke, %d Linien)\n",
		out, len(b.verts), len(b.tris), len(b.lines))
}
