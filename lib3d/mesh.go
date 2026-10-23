package lib3d

import "math"

// ====================================================================
// MESH - reine Zeichen-Geometrie (keine Physik)
//
// Triangles sind xyz-Tripel einer Dreiecksliste, Lines xyz-Tripel einer
// Linienliste (Endpunktpaare). Die Koordinaten sind lokal (Objektraum); die
// Platzierung passiert beim Zeichnen ueber die Modell-Matrix.
// ====================================================================

// Mesh ist reine Zeichen-Geometrie.
type Mesh struct {
	Triangles []float32
	Lines     []float32
}

func addTri(out []float32, a, b, c Vec3) []float32 {
	return append(out,
		float32(a.X), float32(a.Y), float32(a.Z),
		float32(b.X), float32(b.Y), float32(b.Z),
		float32(c.X), float32(c.Y), float32(c.Z),
	)
}

// addQuad trianguliert ein konvexes Viereck als Faecher (a,b,c) + (a,c,d).
func addQuad(out []float32, a, b, c, d Vec3) []float32 {
	out = addTri(out, a, b, c)
	return addTri(out, a, c, d)
}

func addLine(out []float32, a, b Vec3) []float32 {
	return append(out,
		float32(a.X), float32(a.Y), float32(a.Z),
		float32(b.X), float32(b.Y), float32(b.Z),
	)
}

// MakeCubeMesh erzeugt einen achsenparallelen Wuerfel um den Ursprung.
func MakeCubeMesh(w, h, d float64) Mesh {
	hw, hh, hd := w/2, h/2, d/2

	v := [8]Vec3{
		{-hw, -hh, -hd}, {hw, -hh, -hd}, {hw, hh, -hd}, {-hw, hh, -hd},
		{-hw, -hh, hd}, {hw, -hh, hd}, {hw, hh, hd}, {-hw, hh, hd},
	}

	faces := [6][4]int{
		{0, 3, 2, 1}, // vorne
		{4, 5, 6, 7}, // hinten
		{0, 4, 7, 3}, // links
		{1, 2, 6, 5}, // rechts
		{0, 1, 5, 4}, // unten
		{3, 7, 6, 2}, // oben
	}
	edges := [12][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, // vorne
		{4, 5}, {5, 6}, {6, 7}, {7, 4}, // hinten
		{0, 4}, {1, 5}, {2, 6}, {3, 7}, // Verbindungen
	}

	var m Mesh
	for _, f := range faces {
		m.Triangles = addQuad(m.Triangles, v[f[0]], v[f[1]], v[f[2]], v[f[3]])
	}
	for _, e := range edges {
		m.Lines = addLine(m.Lines, v[e[0]], v[e[1]])
	}
	return m
}

// MakePyramidMesh erzeugt eine Pyramide mit quadratischer Basis. Die Basis
// liegt bei y=0 (Mittelpunkt im Ursprung), die Spitze bei y=height.
func MakePyramidMesh(base, height float64) Mesh {
	hb := base / 2

	v := [5]Vec3{
		{-hb, 0, -hb}, {hb, 0, -hb}, {hb, 0, hb}, {-hb, 0, hb}, {0, height, 0},
	}

	sides := [4][3]int{
		{0, 1, 4}, {1, 2, 4}, {2, 3, 4}, {3, 0, 4},
	}
	edges := [8][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, // Basis
		{0, 4}, {1, 4}, {2, 4}, {3, 4}, // Seitenkanten
	}

	var m Mesh
	for _, t := range sides {
		m.Triangles = addTri(m.Triangles, v[t[0]], v[t[1]], v[t[2]])
	}
	m.Triangles = addQuad(m.Triangles, v[3], v[2], v[1], v[0]) // Basis (CCW von unten)
	for _, e := range edges {
		m.Lines = addLine(m.Lines, v[e[0]], v[e[1]])
	}
	return m
}

// MakeSphereMesh erzeugt eine Kugel um den Ursprung.
func MakeSphereMesh(radius float64, slices, stacks int) Mesh {
	if slices < 3 {
		slices = 3
	}
	if stacks < 2 {
		stacks = 2
	}

	tau := 2 * math.Pi
	point := func(i, j int) Vec3 {
		theta := (float64(i) / float64(stacks)) * math.Pi
		phi := (float64(j) / float64(slices)) * tau
		r := radius * math.Sin(theta)
		return Vec3{r * math.Cos(phi), radius * math.Cos(theta), r * math.Sin(phi)}
	}

	var m Mesh
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			a := point(i, j)
			b := point(i, j+1)
			c := point(i+1, j+1)
			d := point(i+1, j)
			if i == 0 {
				m.Triangles = addTri(m.Triangles, a, c, d) // Nordpol
			} else if i == stacks-1 {
				m.Triangles = addTri(m.Triangles, a, b, c) // Suedpol
			} else {
				m.Triangles = addQuad(m.Triangles, a, b, c, d)
			}
		}
	}

	// Meridiane (Pol zu Pol).
	for j := 0; j < slices; j++ {
		for i := 0; i < stacks; i++ {
			m.Lines = addLine(m.Lines, point(i, j), point(i+1, j))
		}
	}
	// Breitenringe (ohne die zusammenfallenden Pol-Ringe).
	for i := 1; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			m.Lines = addLine(m.Lines, point(i, j), point(i, j+1))
		}
	}
	return m
}

// MakeGridMesh erzeugt ein Bodengitter in der XZ-Ebene bei y=0, zentriert um
// den Ursprung. size ist die Kantenlaenge, cells die Anzahl der Zellen pro
// Achse (es entstehen cells+1 Linien je Richtung). Es werden nur Linien
// befuellt (Triangles bleibt leer).
func MakeGridMesh(size float64, cells int) Mesh {
	if cells < 1 {
		cells = 1
	}

	half := size / 2
	step := size / float64(cells)

	var m Mesh
	for i := 0; i <= cells; i++ {
		p := -half + float64(i)*step
		// Linie parallel zur Z-Achse und Linie parallel zur X-Achse.
		m.Lines = addLine(m.Lines, Vec3{p, 0, -half}, Vec3{p, 0, half})
		m.Lines = addLine(m.Lines, Vec3{-half, 0, p}, Vec3{half, 0, p})
	}
	return m
}

// ScaleMesh liefert eine Kopie des Meshes mit allen Koordinaten um den Faktor
// s skaliert (uniform, um den Ursprung). Das Eingabe-Mesh bleibt unveraendert.
//
// Fuer Kollision (Body) das Mesh VOR MakeBody skalieren, damit Geometrie und
// die daraus abgeleiteten Faces konsistent bleiben (Body.modelMatrix() skaliert
// nicht).
func ScaleMesh(mesh Mesh, s float64) Mesh {
	f := float32(s)
	out := Mesh{
		Triangles: make([]float32, len(mesh.Triangles)),
		Lines:     make([]float32, len(mesh.Lines)),
	}
	for i, v := range mesh.Triangles {
		out.Triangles[i] = v * f
	}
	for i, v := range mesh.Lines {
		out.Lines[i] = v * f
	}
	return out
}
