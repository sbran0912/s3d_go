package lib3d

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ====================================================================
// MESH-LADEN (Wavefront OBJ)
//
// Nur das Laden externer Geometrie. Es gibt bewusst KEINE Bau-/Speicher-
// Funktionen in der Bibliothek - Meshes werden ausserhalb erzeugt (Editor
// oder eigenes Skript) und hier eingelesen.
// ====================================================================

// LoadOBJ laedt ein Wavefront-OBJ-Mesh von der Festplatte und wandelt es in
// die flache Mesh-Darstellung der Bibliothek um:
//
//   - "v x y z"        -> Vertex-Pool
//   - "f ..."          -> Triangles (Dreiecke 1:1, Polygone per Fan-Triangulierung)
//   - "l ..."          -> Lines (aufeinanderfolgende Paare)
//
// Kommentare (#) und unbekannte Direktiven (vn, vt, g, o, s, mtllib, usemtl,
// ...) werden ignoriert. Indizes duerfen 1-basiert und negativ sein; die
// a/b/c-Schreibweise (Vertex/UV/Normal) wird akzeptiert, UV/Normal-Teile
// bleiben unberuecksichtigt.
//
// Die Koordinaten bleiben unveraendert (OBJ ist typischerweise Y-up, was zu
// "+Y zeigt nach oben" passt). Die gelieferten Dreiecke koennen direkt fuer
// MakeBody (Faces/Kollision) verwendet werden.
func LoadOBJ(path string) (Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Mesh{}, err
	}
	return parseOBJ(string(data))
}

// parseOBJ parst den OBJ-Text in ein Mesh.
func parseOBJ(data string) (Mesh, error) {
	var verts []Vec3
	var mesh Mesh

	// Zeilenumbrueche normalisieren (Windows CRLF).
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				return Mesh{}, fmt.Errorf("OBJ Zeile %d: unvollstaendiger Vertex", lineNo)
			}
			x, errX := strconv.ParseFloat(fields[1], 64)
			y, errY := strconv.ParseFloat(fields[2], 64)
			z, errZ := strconv.ParseFloat(fields[3], 64)
			if errX != nil || errY != nil || errZ != nil {
				return Mesh{}, fmt.Errorf("OBJ Zeile %d: ungueltiger Vertex", lineNo)
			}
			verts = append(verts, Vec3{X: x, Y: y, Z: z})

		case "f":
			idx, err := objIndices(fields[1:], len(verts))
			if err != nil {
				return Mesh{}, fmt.Errorf("OBJ Zeile %d: %v", lineNo, err)
			}
			// Fan-Triangulierung: (0, j, j+1) fuer j = 1 .. n-2.
			for j := 1; j+1 < len(idx); j++ {
				a, b, c := verts[idx[0]], verts[idx[j]], verts[idx[j+1]]
				mesh.Triangles = append(mesh.Triangles,
					float32(a.X), float32(a.Y), float32(a.Z),
					float32(b.X), float32(b.Y), float32(b.Z),
					float32(c.X), float32(c.Y), float32(c.Z),
				)
			}

		case "l":
			idx, err := objIndices(fields[1:], len(verts))
			if err != nil {
				return Mesh{}, fmt.Errorf("OBJ Zeile %d: %v", lineNo, err)
			}
			// Aufeinanderfolgende Paare: (j, j+1).
			for j := 0; j+1 < len(idx); j++ {
				a, b := verts[idx[j]], verts[idx[j+1]]
				mesh.Lines = append(mesh.Lines,
					float32(a.X), float32(a.Y), float32(a.Z),
					float32(b.X), float32(b.Y), float32(b.Z),
				)
			}

		default:
			// vn, vt, g, o, s, mtllib, usemtl, ... ignorieren.
		}
	}

	return mesh, nil
}

// objIndices parst Face-/Line-Tokens ("i", "i/j", "i/j/k", "i//k") in
// 0-basierte Vertex-Indizes. vertexCount ist die Anzahl der bisher gelesenen
// Vertices (fuer negative Indizes und die Bereichspruefung).
func objIndices(tokens []string, vertexCount int) ([]int, error) {
	idx := make([]int, 0, len(tokens))
	for _, tok := range tokens {
		s := tok
		if slash := strings.IndexByte(s, '/'); slash >= 0 {
			s = s[:slash] // UV/Normal-Teil abschneiden
		}
		if s == "" {
			return nil, fmt.Errorf("ungueltiger Index %q", tok)
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("ungueltiger Index %q", tok)
		}
		if n < 0 {
			n = vertexCount + n // negativ: relativ zum Ende
		} else {
			n = n - 1 // OBJ ist 1-basiert
		}
		if n < 0 || n >= vertexCount {
			return nil, fmt.Errorf("Index %q ausserhalb des Bereichs", tok)
		}
		idx = append(idx, n)
	}
	return idx, nil
}
