package lib3d

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempOBJ legt einen OBJ-Text in einem Test-Temp-Verzeichnis ab und
// liefert den Pfad.
func writeTempOBJ(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.obj")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("temp OBJ schreiben: %v", err)
	}
	return path
}

func TestLoadOBJTrianglesAndQuads(t *testing.T) {
	// Dreieck + Quad (Fan) + eine Linienkette (2 Segmente).
	obj := `
# Kommentar
v 0 0 0
v 1 0 0
v 1 0 1
v 0 0 1
f 1 2 3
f 1 2 3 4
l 1 2 3
`
	m, err := LoadOBJ(writeTempOBJ(t, obj))
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}
	// 1 Dreieck + 2 (Quad-Fan) = 3 Dreiecke -> 9*3 Floats.
	if len(m.Triangles) != 3*9 {
		t.Errorf("Triangles: erwartet %d, got %d", 3*9, len(m.Triangles))
	}
	// Linienkette 1-2-3 -> 2 Segmente -> 2*6 Floats.
	if len(m.Lines) != 2*6 {
		t.Errorf("Lines: erwartet %d, got %d", 2*6, len(m.Lines))
	}
}

func TestLoadOBJIndexForms(t *testing.T) {
	// Negative Indizes und a/b/c- bzw. a//c-Schreibweise.
	obj := `
v 0 0 0
v 2 0 0
v 0 0 2
f -3 -2 -1
v 0 2 0
f 1/10 2/20 4//30
`
	m, err := LoadOBJ(writeTempOBJ(t, obj))
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}
	// 2 Dreiecke.
	if len(m.Triangles) != 2*9 {
		t.Errorf("Triangles: erwartet %d, got %d", 2*9, len(m.Triangles))
	}
	// Erster Vertex des zweiten Dreiecks = v1 = (0,0,0).
	if m.Triangles[9] != 0 || m.Triangles[10] != 0 || m.Triangles[11] != 0 {
		t.Errorf("Vertex-Mapping falsch: %v", m.Triangles[9:12])
	}
}

func TestLoadOBJIgnoresUnknownDirectives(t *testing.T) {
	obj := `
mtllib foo.mtl
o Objekt
g Gruppe
s off
v 0 0 0
v 1 0 0
v 0 1 0
vn 0 0 1
vt 0 0
usemtl rot
f 1 2 3
`
	m, err := LoadOBJ(writeTempOBJ(t, obj))
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}
	if len(m.Triangles) != 9 {
		t.Errorf("Triangles: erwartet 9, got %d", len(m.Triangles))
	}
}

func TestLoadOBJIndexOutOfRange(t *testing.T) {
	obj := `
v 0 0 0
v 1 0 0
f 1 2 9
`
	if _, err := LoadOBJ(writeTempOBJ(t, obj)); err == nil {
		t.Error("erwartet Fehler fuer Index ausserhalb des Bereichs")
	}
}

func TestLoadOBJMissingFile(t *testing.T) {
	if _, err := LoadOBJ(filepath.Join(t.TempDir(), "gibt-es-nicht.obj")); err == nil {
		t.Error("erwartet Fehler fuer fehlende Datei")
	}
}

// TestLoadOBJRocketAsset laedt die generierte Rakete (falls vorhanden) und
// prueft, dass Dreiecke und Linien ankommen.
func TestLoadOBJRocketAsset(t *testing.T) {
	path := filepath.Join("..", "assets", "rocket.obj")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("Asset nicht vorhanden: %v", err)
	}
	m, err := LoadOBJ(path)
	if err != nil {
		t.Fatalf("LoadOBJ(rocket): %v", err)
	}
	if len(m.Triangles) == 0 {
		t.Error("Rakete: keine Dreiecke geladen")
	}
	if len(m.Lines) == 0 {
		t.Error("Rakete: keine Linien geladen")
	}
}
