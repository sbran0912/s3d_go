# Bibliothek lib3d – Referenz (Go)

`lib3d` ist eine kleine 3D-Zeichenbibliothek auf OpenGL 3.3 (GLFW), geschrieben
in Go. Sie stellt Szenen-/Kamera-Bündel, Farb-, Mathe- und Mesh-Bausteine bereit
und sammelt alle Zeichenaufrufe eines Frames zu effizienten Batches. Sie
enthält außerdem Körper (`Body`) mit Flächen/Normalen und Schnittabfragen
(Raycasting), zeichnet aber selbst nicht.

- Go-Modul: `s3dgo`
- Importpfad der Bibliothek: `s3dgo/lib3d` (Paketname: `lib3d`)
- Aufbau/Build: siehe [`docs/build.md`](build.md)

Namenskonvention (Port aus dem TypeScript-Original): Typen in `PascalCase`,
Funktionen/Methoden in `PascalCase`, Felder in `PascalCase` (exportiert). Nur
renderer-interne Helfer sind nicht exportiert.

Import-Beispiel:

```go
import (
    "s3dgo/lib3d"
)
```

---

## 1. Grundgerüst eines Programms

```go
package main

import (
    "fmt"
    "os"

    "s3dgo/lib3d"
)

func main() {
    scene := lib3d.NewScene() // Kamera + Welt-Attribute (Defaults s.u.)

    draw := func() {
        lib3d.BeginScene(scene) // Background, Nebel, Kamera, Licht
        // ... hier zeichnen ...
    }

    if err := lib3d.Init(); err != nil { // Fenster + GL-Kontext + Shader
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    lib3d.StartAnimation(draw) // Animations-Schleife, ruft draw pro Frame
}
```

**Ablauf pro Frame:** `StartAnimation` setzt die Zeit-Uniform, ruft die
übergebene `draw`-Funktion auf und zeichnet danach automatisch alle im Frame
gesammelten Zeichenbefehle (`FlushBatch`). Man zeichnet also *nur* über die
`Draw*`-Funktionen; das eigentliche Rendern/Austauschen der Puffer
übernimmt die Bibliothek.

| Funktion | Bedeutung |
| --- | --- |
| `Init() error` | Fenster im Fenstermodus (zentriert, 1280×800), OpenGL-Kontext, Shader, Batch-Zustand. |
| `StartAnimation(fnDraw func())` | Startet die Schleife; `fnDraw` wird jedes Frame aufgerufen. |
| `NoLoop()` | Beendet die Animations-Schleife. |
| `ToggleFullscreen()` | Schaltet zwischen Fenster- und Vollbildmodus um. |
| `SetFullscreen(on bool)` | Setzt den Vollbildmodus explizit. |
| `IsFullscreen() bool` | `true`, wenn das Fenster im Vollbildmodus ist. |
| `GetWidth()` / `GetHeight()` | Fenstergröße in Pixeln. |
| `IsMouseDown()` | `true`, solange linke Maustaste gedrückt. |
| `IsMouseUp()` | einmalig `true` beim Loslassen. |
| `IsKeyDown(key Key)` | `true`, solange die Taste gedrückt ist (Dauerdruck). |
| `IsKeyPressed(key Key)` | einmalig `true` beim Drücken (konsumierend, für Trigger). |

> **Fenster, Vollbild & Beenden:** `Init` öffnet ein normales Fenster
> (zentriert, 1280×800; auf kleinen Monitoren automatisch begrenzt). Mit
> **F11** wird zur Laufzeit zwischen Fenster- und Vollbildmodus umgeschaltet –
> das erledigt die Bibliothek selbst, alle Apps bekommen es automatisch.
> Programmatisch geht es über `ToggleFullscreen()` bzw. `SetFullscreen(on)`;
> `IsFullscreen()` fragt den Status ab. Der **OpenGL-Kontext bleibt beim
> Umschalten erhalten** (kein Fenster-Neuaufbau); Fenstergröße, Viewport und
> Aspect aktualisieren sich automatisch. Beendet wird mit **ESC** oder Alt+F4;
> die Schleife läuft aus und `glfwTerminate()` stellt den vorherigen
> Video-Modus wieder her. `NoLoop()` stoppt die Schleife ebenso sauber.

> **Hinweis (Go/GLFW):** `Init` ruft intern `runtime.LockOSThread()` auf, muss
> also aus der `main`-Goroutine aufgerufen werden. Alle Fenster-/GL-Aufrufe
> laufen dann im selben OS-Thread.

### Tastatureingaben (Polling)

Tastatur wird **pollend** im `draw`-Closure abgefragt. **ESC** (Beenden) und
**F11** (Vollbild-Umschaltung) sind fest in der Bibliothek belegt.

- `IsKeyDown(key)` – `true`, solange die Taste gedrückt ist → für
  **laufende Steuerung**.
- `IsKeyPressed(key)` – einmalig `true` beim Drücken; der Aufruf
  **konsumiert** die Kante → für **Trigger**.

Die Schlüssel stammen aus dem Typ `lib3d.Key` (`lib3d.KeyLeft`, `lib3d.KeySpace`,
…); die Werte entsprechen den `GLFW_KEY_*`-Codes, daher kann auch ein roher
GLFW-Code über `lib3d.Key(glfwCode)` übergeben werden.

```go
const speed = 0.03
if lib3d.IsKeyDown(lib3d.KeyLeft) {
    camAngle -= speed
}
if lib3d.IsKeyPressed(lib3d.KeySpace) {
    lib3d.SetEffect(lib3d.EffectPulse)
}
```

> Hinweis: `IsKeyPressed` liefert die Kante nur **einmal** – wer denselben
> Tastendruck in mehreren Zweigen auswerten will, liest sie in eine Variable.

---

## 2. Koordinatensystem, Kamera und Szene

**Koordinatensystem:** `+Y` zeigt nach oben, die Kamera blickt in `+Z`.
View- und Projection-Matrix laufen im Vertex-Shader; die Modell-Matrix wird
pro Zeichenaufruf als Argument übergeben (kein globaler Modell-Zustand).

```go
type Camera struct {
    Pos    lib3d.Vec3 // Augpunkt (Weltkoordinaten)
    Target lib3d.Vec3 // Blickziel (Weltkoordinaten)
    Up     lib3d.Vec3 // Oben-Vektor (Weltkoordinaten)
    FovY   float64    // vertikales Sichtfeld (Bogenmaß)
    Near   float64    // nahe Clipping-Ebene
    Far    float64    // ferne Clipping-Ebene
}

func (c Camera) View() lib3d.Matrix4x4                  // Look-At-Matrix
func (c Camera) Projection(aspect float64) lib3d.Matrix4x4 // Perspective

type Scene struct {
    Cam        Camera
    SunDir     lib3d.Vec3      // Lichtrichtung (Weltraum)
    SceneFog   Fog             // Tiefen-Nebel
    Background lib3d.ColorState // Fenster-Hintergrund (RGBA, 0..1)
}

type Fog struct {
    Enabled bool
    Near    float64
    Far     float64
    Color   lib3d.ColorState
}
```

`NewScene()` liefert die Defaults: Kamera `Pos{50,100,200}` / `Target{0,0,0}` /
`Up{0,1,0}` / `FovY 1.2` / `Near 0.1` / `Far 1000`, `SunDir{0.5,1,0.3}`,
Nebel `Enabled` `100..400`, dunkelgrauer Hintergrund.

### 2.1 Die Attribute von `Camera`

`Pos` und `Target` bilden ein **Paar** – die Kamera schaut von `Pos` nach
`Target`.

| Attribut | Einheit | Bedeutung | Muss-Bedingung |
|---|---|---|---|
| `Pos` | Weltkoordinaten | Augpunkt. | `Pos != Target` |
| `Target` | Weltkoordinaten | Blickziel (nur Richtung, nicht Entfernung). | `Pos != Target` |
| `Up` | Weltkoordinaten | Referenz-„Oben". | nicht parallel zu `Target - Pos` |
| `FovY` | **Bogenmaß** | Vertikales Blickfeld. | `0 < FovY < π` |
| `Near` | Kamera-Einheiten | Vordere Clipping-Ebene. | `Near > 0` |
| `Far` | Kamera-Einheiten | Hintere Clipping-Ebene. | `Far > Near` |

**`Pos` und `Target`:** `Target` ist **keine Position**, sondern nur eine
Richtungsangabe. Für eine **Orbit-Kamera** bleibt `Target` fest und nur `Pos`
wandert auf einem Kreis darum herum.

**`Up`:** Muss nicht senkrecht auf der Blickrichtung stehen. `LookAtMatrix`
orthonormalisiert automatisch (Gram-Schmidt). Typisch `{0,1,0}`. `Up` darf nicht
parallel zur Blickrichtung sein.

**`FovY`:** Bogenmaß, nicht Grad. Default `1.2 rad` ≈ `68.8°`.

### 2.2 Die Attribute von `Scene`

| Attribut | Bedeutung |
|---|---|
| `Cam` | Kamera der Szene. Wird von `BeginScene` via `ApplyCamera` gesetzt. |
| `SunDir` | Sonnenrichtung in **Weltkoordinaten**; `BeginScene` transformiert sie in Kamerakoordinaten und ruft `SetLightDirection`. |
| `SceneFog` | Tiefen-Nebel. `Enabled == false` → kein Nebel. |
| `Background` | Füllfarbe vor dem Zeichnen (RGBA, 0..1). |

- `BeginScene(scene)` – pro Frame aufrufen.
- `ApplyCamera(cam)` – setzt View + Projection (Aspect aus `GetWidth/GetHeight`).
- `SetView(m)` / `SetProjection(m)` – manuelle Kamera-Matrizen.
- `SetFog(near, far, r, g, b, a)` – Tiefen-Nebel direkt.
- `SetLightDirection(x, y, z)` – Lichtrichtung (Kamerakoordinaten).
- `SetBatchSort(bool)` – fasst auch nicht-benachbarte Primitive zusammen.
  **Achtung:** nur für opake Geometrie. `CompactBatch()` sortiert/fasst manuell.
- `GetPendingDrawCalls()` / `GetPendingInstancedDrawCalls()` – Diagnose.

---

## 3. Farben

```go
type ColorState struct { R, G, B, A float64 } // Werte 0..1
```

`ColorInput` kapselt eine optionale Farbangabe (Felder in `PrimitiveStyle` /
`ShapeStyle`). Der Nullwert (`Set == false`) überschreibt den aktuellen
Low-Level-Zustand **nicht**. In Go fehlt Overloading, daher erzeugen Helfer einen
gesetzten `ColorInput`:

```go
lib3d.Hex("#ff8800")          // Hex (#rgb oder #rrggbb)
lib3d.Gray(128)               // Graustufe 0..255
lib3d.RGB(255, 128, 0)        // r,g,b 0..255
lib3d.RGBA(255, 128, 0, 200)  // r,g,b,a 0..255
lib3d.ColorOf(colorState)     // aus einem ColorState
```

Parser (liefern `ColorState` mit 0..1):

```go
lib3d.ParseHex("#44aaff")
lib3d.ParseGray(128)
lib3d.ParseColorRGB(255, 128, 0)
lib3d.ParseColorRGBA(255, 128, 0, 200)
lib3d.ColEq(a, b) // exakter Vergleich
```

---

## 4. Objekte erstellen

Es gibt zwei Wege: **ohne Body** (nur Zeichengeometrie) und **mit Body**
(Mesh + lokale Flächen/Normalen für Kollision/Raycasting). In beiden Fällen
wird die Geometrie als `Mesh` erzeugt und über `DrawMesh` gezeichnet.

```go
type Mesh struct {
    Triangles []float32 // xyz-Tripel je Dreieck (9 Floats/Dreieck)
    Lines     []float32 // xyz-Tripel je Linien-Endpunkt (Paare)
}

func MakeCubeMesh(w, h, d float64) Mesh
func MakePyramidMesh(base, height float64) Mesh // Basis y=0, Spitze y=height
func MakeSphereMesh(radius float64, slices, stacks int) Mesh
func MakeGridMesh(size float64, cells int) Mesh // Bodengitter, XZ-Ebene y=0
```

> `MakeGridMesh(size, cells)` – `size` = Kantenlänge, `cells` = Zellen pro
> Achse (es entstehen `cells+1` Linien je Richtung). Es füllt nur `Lines`.

### 4.1 Objekt **ohne** Body

```go
cubeStyle := lib3d.ShapeStyle{
    Stroke:    lib3d.Hex("#1b1b1b"),
    Fill:      lib3d.Hex("#d3ff44"),
    LineWidth: 1,
    Mode:      lib3d.StyleBoth,
}

cubeMesh := lib3d.MakeCubeMesh(30, 30, 30) // EINMAL erzeugen

angle := 0.0
draw := func() {
    lib3d.BeginScene(scene)
    angle -= 0.01

    model := lib3d.MultMatrix(lib3d.RotateMatrix(0, angle, 0),
        lib3d.TranslateMatrix(-45, 15, 0))
    lib3d.DrawMesh(cubeMesh, cubeStyle, model)
}
```

Modell-Matrix-Helfer: `IdentityMatrix()`, `TranslateMatrix(dx,dy,dz)`,
`RotateMatrix(ax,ay,az)`, `MultMatrix(a,b)` (`a*b`) und `ViewWorldMatrix`.

### 4.2 Objekt **mit** Body

```go
type Body struct { // Paket lib3d
    Geometry Mesh
    Faces    []Face
    Pos      Vec3
    Rot      Vec3 // Euler-Winkel in Radiant (Rz*Ry*Rx)
}

func (b Body) ModelMatrix() Matrix4x4 // Translation * Rotation
func (b Body) WorldFaces() []Face     // Faces in Weltkoordinaten

func MakeBody(geometry Mesh, pos Vec3, rot ...Vec3) Body
```

```go
ballMesh := lib3d.MakeSphereMesh(40, 40, 40)
ballBody := lib3d.MakeBody(ballMesh, lib3d.Vec3{X: 0, Y: 40, Z: 0}) // Faces automatisch

angle := 0.0
draw := func() {
    lib3d.BeginScene(scene)
    angle += 0.01
    ballBody.Rot.Y = angle
    lib3d.DrawMesh(ballBody.Geometry, ballStyle, ballBody.ModelMatrix())
}
```

| | ohne Body | mit Body |
| --- | --- | --- |
| Typ | `lib3d.Mesh` | `lib3d.Body` (enthält `Mesh`) |
| Erzeugen | `Make*Mesh(...)` | `lib3d.MakeBody(mesh, pos, vec3(rotX, rotY, rotZ))` |
| Zeichnen | `DrawMesh(mesh, style, model)` | `DrawMesh(body.Geometry, style, body.ModelMatrix())` |
| Rotation | Modell-Matrix | `body.Rot` (Euler-Vec3) + `ModelMatrix()` |
| Kollision/Raycast | nicht möglich | `body.Faces`, `body.WorldFaces()`, `RaycastSegment` |

### 4.3 Externe Meshes laden (OBJ)

Die Bibliothek kann Meshes im **Wavefront-OBJ**-Format von der Festplatte
laden. Es gibt bewusst **keine** Bau-/Speicher-Funktionen – Meshes werden
außerhalb erzeugt (Editor wie Blender oder ein eigenes Skript) und hier
eingelesen.

```go
func LoadOBJ(path string) (Mesh, error)
```

Parser-Umfang:

- `v x y z` → Vertex-Pool
- `f …` → `Triangles` (Dreiecke 1:1, Quads/Polygone per Fan-Triangulierung)
- `l …` → `Lines` (aufeinanderfolgende Paare)
- Indizes **1-basiert und negativ**; `a/b/c` bzw. `a//c` erlaubt (UV/Normal-Teile
  werden ignoriert)
- Kommentare `#` und unbekannte Direktiven (`vn`, `vt`, `g`, `o`, `s`,
  `mtllib`, `usemtl`) werden übersprungen

```go
vehicleMesh, err := lib3d.LoadOBJ("rocket.obj")
if err != nil {
    log.Fatal(err)
}
// direkt weiterverwenden wie jedes Make*Mesh:
lib3d.DrawMesh(vehicleMesh, style, model)
body := lib3d.MakeBody(vehicleMesh, lib3d.Vec3{X: 0, Y: 0, Z: 0})
```

Hinweise:
- Koordinaten bleiben unverändert (OBJ ist typischerweise **Y-up**, passend zu
  „+Y zeigt nach oben").
- Für Drahtgitter müssen im OBJ **`l`-Linien** enthalten sein (Styles sind oft
  `StyleBoth`).
- Für Kollision (`Body`) das Mesh in **Endgröße** laden – nicht nur per
  Modell-Matrix skalieren (sonst passen `WorldFaces()` und Rendering nicht
  zusammen). Mehrere OBJ-Objekte/Gruppen werden zu **einem** `Mesh`
  zusammengeführt.

#### Mesh skalieren (geladene Meshes)

Zum Vergrößern/Verkleinern eines Meshes gibt es `lib3d.ScaleMesh`. Es liefert
eine **Kopie** mit allen Koordinaten (Triangles und Lines) um den Faktor `s`
skaliert – uniform und um den Ursprung; das Eingabe-Mesh bleibt unverändert.

```go
func ScaleMesh(mesh Mesh, s float64) Mesh

mesh, _ := lib3d.LoadOBJ("rocket.obj")
body := lib3d.MakeBody(lib3d.ScaleMesh(mesh, 2.0), lib3d.Vec3{})
```

**Wichtig:** Das Mesh **vor** `MakeBody` skalieren (Skalierung der Mesh-Daten),
damit `Faces`/`WorldFaces()` zur Geometrie passen. Eine Skalierung *nur* über
die Modell-Matrix würde `Body` nicht erfassen, weil `Body.ModelMatrix()` nur
`Translation · Rotation` rechnet. Eine Skalierungs-Matrix (`scaleMatrix`) gibt
es bewusst nicht.

Die App `rocket` nutzt `lib3d.ScaleMesh(mesh, rocketScale)`.

#### Blender-Export (Hinweise)

- **Normalen nicht nötig:** `vn` (und `vt`) werden ignoriert. Auch wenn die
  `f`-Zeilen sie referenzieren (z. B. `f 1//1 2//2 3//3`), wird nur der Teil vor
  dem ersten `/` gelesen. Die Normalen in Blender also einfach anlassen.
- **Triangulate Faces** aktivieren → saubere Dreiecke (sonst Fan-Triangulierung
  von Quads/ngons).
- **Write Edges / Include Edges** aktivieren → schreibt `l`-Linien; nur so
  bekommst du das Drahtgitter (Styles sind oft `StyleBoth`).
- **Achsen** mit den Defaults belassen (`Forward -Z`, `Up Y`) – passt zu
  „+Y zeigt nach oben".
- Objekte/Gruppen sowie `usemtl`/`mtllib` sind egal – alles wird zu **einem**
  `Mesh` zusammengeführt.
- **Shading bleibt flach:** Auch bei „Shade Smooth" leuchtet die Bibliothek pro
  Dreiecksfläche aus (Normale aus `dFdx`/`dFdy`).
- **Für Kollision (`Body`):** In Blender die Normalen konsistent nach außen richten
  („Recalculate Outside") – die Kollision leitet die Flächennormale aus der
  Windungsreihenfolge ab.

---

## 5. Rendermethoden

Alle Zeichenaufrufe nehmen ihre **Modell-Matrix als letztes (optionales)
Argument** (Default Identität) und sammeln sich im aktuellen Batch.

### 5.1 Zeichenmodi und Style-Objekte

```go
const ( // lib3d.DrawStyle
    StyleStroke = lib3d.StyleStroke // 0
    StyleFill   = lib3d.StyleFill   // 1
    StyleBoth   = lib3d.StyleBoth   // 2
)

type PrimitiveStyle struct { // für Punkte/Linien
    Stroke    ColorInput // Linienfarbe (unbeleuchtet)
    Fill      ColorInput // Füllfarbe (beleuchtet)
    LineWidth float64    // 0 = Zustand nicht ändern
    PointSize float64    // 0 = Zustand nicht ändern
}

type ShapeStyle struct { // für Flächen/Primitive mit Modus
    Stroke, Fill ColorInput
    LineWidth, PointSize float64
    Mode     DrawStyle // was gezeichnet wird
    Z        float64   // Z-Ebene für Rect
    Segments int       // Segmentzahl für Circle (0 -> 64)
}
```

Nicht gesetzte Felder (`ColorInput.Set == false` bzw. `0`) überschreiben den
aktuellen Low-Level-Zustand **nicht** – man kann Stile schrittweise vererben.

### 5.2 Übersicht der Renderfunktionen

| Funktion | Beschreibung |
| --- | --- |
| `DrawMesh(mesh, style, model…)` | Mesh mit Fill (beleuchtet) und/oder Drahtgitter-Linien. |
| `DrawMeshInstanced(mesh, style, instances)` | **GPU-Instancing**: dieselbe Geometrie in *einem* Draw-Call. |
| `DrawPoint(x,y,z, PrimitiveStyle, model…)` | Einzelner Punkt (`PointSize`). |
| `DrawLine(x1…z2, PrimitiveStyle, model…)` | Einzelne Linie. |
| `DrawTriangle(x1…z3, ShapeStyle, model…)` | Dreieck (Fill/Stroke/Both). |
| `DrawShape(x1…z4, ShapeStyle, model…)` | Viereck aus 4 Punkten. |
| `DrawRect(x,y,w,h, ShapeStyle, model…)` | Rechteck in der X/Y-Ebene auf `style.Z`. |
| `DrawCircle(x,y,z, radius, ShapeStyle, model…)` | Kreis (`style.Segments`; 0 → 64). |
| `DrawPolygon(pts []float64, ShapeStyle, model…)` | Konvexes Polygon (xyz-Tripel). |

**GPU-Instancing** (`Instance`):

```go
type Instance struct {
    Model Matrix4x4
    Color ColorState
}

cubes := make([]lib3d.Instance, n)
for i := range cubes {
    cubes[i].Model = lib3d.MultMatrix(lib3d.TranslateMatrix(px, py, pz),
        lib3d.RotateMatrix(0, angle, 0))
    cubes[i].Color = lib3d.ColorState{0.25 + 0.6*t, 0.45, 0.9 - 0.6*t, 1}
}
lib3d.DrawMeshInstanced(cubeMesh, cubeStyle, cubes) // EIN Draw-Call für alle
```

> Beim Instancing wird die Farbe aus dem `ShapeStyle` ignoriert; sie kommt aus
> jeder `Instance`.

### 5.3 Low-Level-Zustand und Effekte

| Zustand | Setter | Default |
| --- | --- | --- |
| Füllfarbe | `FillColor(ColorState)` / `FillColorInput(ColorInput)` | weiß |
| Linienfarbe | `StrokeColor(ColorState)` / `StrokeColorInput(ColorInput)` | schwarz |
| Linienbreite | `StrokeWidth(float64)` | 1 |
| Punktgröße | `PointSize(float64)` | 4 |
| Effekt | `SetEffect(EffectMode)` | `EffectFlat` |
| Gradient-Endfarbe | `SetGradient(ColorState)` / `SetGradientInput` | – |
| Gradient-Zentrum/Radius | `SetGradientCenter(cx,cy,cz,radius)` | – |
| Hintergrund | `Background(ColorState)` / `BackgroundInput` | – |

Primitive im Low-Level-Stil: `Point`, `Line`, `Triangle`, `Shape`, `Rect`,
`Circle`, `Polygon` – gleiche Argumente wie die `Draw*`-Variante, aber ohne
Style-Objekt.

**Shader-Effekte** (`EffectMode`):

| Wert | Effekt |
| --- | --- |
| `EffectFlat` (0) | einfache Farbe (Default). |
| `EffectGradient` (1) | radialer Verlauf zur `SetGradient`-Farbe vom `SetGradientCenter`. |
| `EffectPulse` (2) | sinusförmige Helligkeitspulsation (nutzt die Frame-Zeit). |

---

## 6. Körper, Flächen, Normalen, Raycasting

```go
type Face struct { A, B, C, Normal Vec3 } // Dreieck + normierte Normale

func BuildFaces(mesh Mesh) []Face
func TransformFace(face Face, model Matrix4x4) Face
func TransformFaces(faces []Face, model Matrix4x4) []Face

type RayHit struct {
    IsHit  bool
    T      float64 // Parameter entlang der Strecke [0..1]
    Point  Vec3    // Trefferpunkt
    Normal Vec3    // Flächennormale am Treffer
}
func RaycastSegment(p1, p2 Vec3, faces []Face) RayHit

type ClosestFaceHit struct {
    Point          Vec3
    Normal         Vec3
    Distance       float64 // |Point - p| (unsigned)
    SignedDistance float64 // >0 außen, <0 innen (bzgl. Normale)
}
func ClosestPointToFace(p Vec3, face Face) ClosestFaceHit
```

Typischer Einsatz (Strahlen gegen alle Körper):

```go
var worldFaces []lib3d.Face
for _, b := range sceneBodies {
    worldFaces = append(worldFaces, b.Object.WorldFaces()...)
}

hit := lib3d.RaycastSegment(apex, end, worldFaces)
endpoint := end
if hit.IsHit {
    endpoint = hit.Point // Strahl am Treffer kappen
}

lib3d.DrawLine(apex.X, apex.Y, apex.Z, endpoint.X, endpoint.Y, endpoint.Z,
    lib3d.PrimitiveStyle{Stroke: lib3d.Hex("#ff8800"), LineWidth: 1})
```

---

## 7. Mathe-Helfer

**`Vec3`** (`X, Y, Z`): `Add`, `Sub`, `Scale`, `Mag(m)`, `Limit(max)`, `Negate`,
`Dot`, `Cross`, `SquaredLength`, `Length`, `DistanceTo`, `Normalize`,
`Lerp(v,t)`, `Clone`, `Equals(v,eps)`, `Transform(Matrix4x4)`,
`TransformDir(Matrix4x4)`.

**`Matrix4x4`** – `[4][4]float64`, **row-major** (intern column-major für
OpenGL): `TranslateMatrix`, `RotateMatrix`, `MultMatrix`, `IdentityMatrix`,
`ViewWorldMatrix`, `LookAtMatrix` (blickt in `+Z`), `WorldToCamera`,
`RotateAround(point, pivot, rotation)`,
`PerspectiveMatrix(fovY, aspect, near, far)`.

**`math.go`**: `Project(fov, Vec3)`, `Constrain(value, lo, hi)`,
`Random(n1, n2)` (int, `[n1,n2)`), `RandomFloat(lo, hi)` (float64, `[lo,hi)`).

---

## 8. Vollständiges Minimalbeispiel (mit Body)

```go
package main

import (
    "fmt"
    "os"

    "s3dgo/lib3d"
)

func main() {
    scene := lib3d.NewScene()
    scene.Cam.Pos = lib3d.Vec3{X: 50, Y: 100, Z: 200}
    scene.SunDir = lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
    scene.SceneFog = lib3d.Fog{Enabled: true, Near: 100, Far: 400,
        Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

    ballStyle := lib3d.ShapeStyle{
        Stroke:    lib3d.Hex("#2f3a00"),
        Fill:      lib3d.Hex("#d3ff44"),
        LineWidth: 1,
        Mode:      lib3d.StyleBoth,
    }

    ball := lib3d.MakeBody(lib3d.MakeSphereMesh(40, 40, 40), lib3d.Vec3{X: 0, Y: 40, Z: 0})
    angle := 0.0

    draw := func() {
        lib3d.BeginScene(scene)
        angle += 0.01
        ball.Rot.Y = angle
        lib3d.DrawMesh(ball.Geometry, ballStyle, ball.ModelMatrix())

        // Kollision/Raycasting-Beispiel: Abstand zur Oberfläche
        faces := ball.WorldFaces()
        _ = lib3d.ClosestPointToFace(lib3d.Vec3{}, faces[0])
    }

    if err := lib3d.Init(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    lib3d.SetBatchSort(true)
    lib3d.StartAnimation(draw)
}
```

Weitere vollständige Beispiele liegen unter `apps/`:
`beispiel` (Meshes + Szenenrotation), `vorlage` (Body + Gitter),
`instanzen` (GPU-Instancing), `ray` (Body-Faces + Raycasting + Effekte),
`boids` (Schwarm mit Vehicle/Steering),
`rocket` (wie `boids`, aber die Vehicles sind ein **geladenes** OBJ aus
`apps/rocket/rocket.obj`; Hindernis-Würfel und Gitter bleiben prozedural),
`vehic` (Fahrzeug-Schwarm mit Seek/DNA: frisst gutes Futter und Gift, Health
und Altern; Futter und Fahrzeuge werden per **GPU-Instancing** gezeichnet).

---

## 9. Hinweise & Fallstricke

- **Keine globale Modell-Matrix:** Position/Rotation immer als `model`-Argument
  bzw. über `body.ModelMatrix()` übergeben.
- **Stile erben:** Nur gesetzte Felder überschreiben den Zustand. Für
  reproduzierbare Ergebnisse im Zweifel `ColorInput` explizit setzen.
- **Batching:** Alle `Draw*`-Aufrufe eines Frames werden gesammelt und am
  Frame-Ende automatisch gezeichnet. `Background(...)` zeichnet den Batch
  vorher aus.
- **`SetBatchSort(true)`** nur für opake Geometrie.
- **Kamera:** `Target` ist nur eine Richtungsangabe. `FovY` in **Bogenmaß**.
  `Near > 0`, `Far > Near`, `Up` nicht parallel zur Blickrichtung.
- **Clipping vs. Nebel:** `Camera.Far` schneidet hart ab, `SceneFog.Far` nebelt
  nur aus.
- **Meshes wiederverwenden:** Geometrie einmal erzeugen und teilen.
- **Body vs. Mesh:** Renderdaten (Farbe/Linienbreite) gehören in den
  `ShapeStyle`, nicht in den `Body`.
- **Go/CGO:** Das Projekt nutzt CGO (`go-gl`); `CGO_ENABLED=1` und ein
  C-Compiler müssen vorhanden sein. `Init` muss aus der `main`-Goroutine
  laufen (LockOSThread erfolgt intern).
