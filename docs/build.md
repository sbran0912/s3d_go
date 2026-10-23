# Build-Prozess (Go)

Dieses Projekt ist das Go-Modul **s3dgo**. Es baut die Bibliothek **lib3d** als
Go-Paket und linkt daraus die Anwendungen in `apps/`.

> Für die Benutzung der Bibliothek (Objekte mit/ohne Body, Rendermethoden)
> siehe [`docs/library.md`](library.md).

## Verzeichnisstruktur

```
.
├── go.mod            Moduldefinition (module s3dgo)
├── go.sum            Abhängigkeits-Prüfsummen
├── Makefile          Build-Regeln (Build, Start, Test)
├── lib3d/            Bibliothek (Paket lib3d)
│   ├── vec.go        Vec3 / Vec2
│   ├── mat4.go       Matrix4x4 + Matrix-Helfer
│   ├── color.go      ColorState / ColorInput / Parse*
│   ├── math.go       Project, Constrain, Random, RandomFloat
│   ├── mesh.go       Mesh + Make*Mesh
│   ├── shader.go     GLSL-Quellen + Programmerzeugung
│   ├── renderer.go   Batching, Init/Loop, Eingabe, Szene, Instancing
│   ├── draw.go       Zeichen-Fassade (Draw*)
│   ├── body.go       Body, Faces, Raycasting
│   ├── body_test.go  Unit-Test für Body/Faces
│   └── lib3d_test.go Unit-Test der Bibliothek
├── apps/             Anwendungen, je App ein Unterordner
│   ├── beispiel/main.go
│   ├── vorlage/main.go
│   ├── instanzen/main.go
│   ├── ray/main.go
│   ├── boids/main.go + vehicle.go
│   └── rocket/main.go + vehicle.go + rocket.obj
└── bin/              fertig gebaute Programme (inkl. kopierter Mesh-Assets)
```

Anders als in der C++-Fassung gibt es keine Objektdateien/statische Bibliothek:
Die Bibliothek ist ein normales Go-Paket (`lib3d`), und die Apps werden direkt
aus dem Quellbaum gebaut.

## Voraussetzungen

- **Go** (getestet mit go1.27) mit aktiviertem CGO (`CGO_ENABLED=1`).
- Ein **C-Compiler** (z. B. `gcc`) – `go-gl` bindet GLFW/GL über CGO.
- **GLFW3 + OpenGL** (`libglfw3-dev`, `libgl-dev`) – vorhanden, wenn
  `pkg-config --exists glfw3 && pkg-config --exists gl`.
- Go-Module `github.com/go-gl/gl/v3.3-core/gl` und
  `github.com/go-gl/glfw/v3.3/glfw` (in `go.mod` festgeschrieben).

## Targets

| Befehl | Wirkung |
| --- | --- |
| `make` / `make build` | Baut **alle** Apps nach `bin/`. |
| `make apps` | Alias für `make build`. |
| `make bin/<name>` | Baut **nur eine** App, z. B. `make bin/ray`. |
| `make run APP=<name>` | Startet die App `<name>` im App-Verzeichnis (`cd apps/<name>` + `go run .`). |
| `make run-<name>` | Kurzform zum Starten, z. B. `make run-ray` (wechselt nach `apps/<name>`). |
| `make vet` | `go vet ./...`. |
| `make test` | `go test ./...`. |
| `make clean` | Löscht `bin/`. |

Verfügbare App-Namen (aus `APP_NAMES` im Makefile):
`beispiel`, `vorlage`, `instanzen`, `ray`, `boids`, `rocket`.

Die App `rocket` (basiert auf `boids`) lädt ihr Vehicle-Mesh zur Laufzeit aus
`rocket.obj` (Wavefront-OBJ, siehe `docs/library.md`, Abschnitt 4.3). Das OBJ
liegt im App-Verzeichnis `apps/rocket/`. `make run-rocket` wechselt vor dem
Start nach `apps/rocket`, und der Build kopiert die Datei nach `bin/` – so liegt
sie immer neben dem laufenden Programm. Das geladene Mesh wird um `rocketScale`
vergrößert (`lib3d.ScaleMesh`, Skalierung der Mesh-Daten – damit bleiben
Geometrie und `Faces` konsistent).

### Beispiele

```bash
make                  # alle Apps bauen
make bin/ray          # nur "ray" bauen
make run APP=boids    # "boids" starten
make run-vorlage      # dasselbe, kurz
make test             # Unit-Tests ausführen
make vet              # statische Analyse
make clean            # aufräumen
```

Wird `make run` ohne `APP` aufgerufen, gibt das Makefile eine Fehlermeldung mit
der Liste der verfügbaren Apps aus.

## Wichtiger Hinweis zum Starten

Die Anwendungen starten im **Fenstermodus** (zentriert, 1280×800) und
benötigen eine laufende grafische Sitzung (X11/Wayland mit GL). Mit **F11**
wird zur Laufzeit auf **Vollbild** umgeschaltet und wieder zurück – das
erledigt die Bibliothek, alle Apps bekommen es automatisch. Programmatisch geht
es über `lib3d.ToggleFullscreen()` / `lib3d.SetFullscreen(on)` /
`lib3d.IsFullscreen()`; der **OpenGL-Kontext bleibt beim Umschalten erhalten**
(kein Fenster-Neuaufbau).

`Init` muss aus der `main`-Goroutine aufgerufen werden (intern wird
`runtime.LockOSThread()` gesetzt). Für echtes Vollbild wird ein Monitor
benötigt; ohne Monitor bleibt das Fenster im Fenstermodus (der Start schlägt
nicht fehl). In einer reinen SSH-/Headless-Sitzung lässt sich das Fenster
weiterhin nicht erzeugen.

## Wie gebaut wird

1. **Bibliothek:** `lib3d` ist ein normales Go-Paket. Es gibt keinen
   separaten Archivierungsschritt – `go build ./...` kompiliert es
   bei Bedarf.
2. **Apps:** Jede App ist ein `main`-Paket unter `apps/<name>/`. Mehrere
   `.go`-Dateien pro App sind möglich (z. B. `boids/main.go` + `boids/vehicle.go`)
   und werden automatisch gemeinsam kompiliert. Gebaut wird mit:

   ```bash
   go build -o bin/<name> ./apps/<name>
   ```

3. **Tests:** `go test ./...` kompiliert und führt alle Paket-Tests aus
   (`lib3d/lib3d_test.go`, `lib3d/mesh_io_test.go`, `lib3d/body_test.go`).

## Erweitern

### Neue Bibliotheks-Quelldatei

1. `.go`-Datei in `lib3d/` ablegen (`package lib3d`).
2. Fertig – Go kompiliert alle Dateien des Pakets automatisch.

### Neue App

1. Ordner `apps/<name>/` mit `main.go` (`package main`) anlegen.
2. Im Makefile in der Zeile `APP_NAMES := …` den Namen ergänzen.
3. Bauen mit `make`, starten mit `make run APP=<name>` bzw. `make run-<name>`.

Optional können im App-Ordner weitere `.go`-Dateien liegen (z. B.
`vehicle.go`); sie werden automatisch mitkompiliert, sobald der App-Name in
`APP_NAMES` steht.

### Externe Meshes (`apps/<name>/*.obj`)

Meshes werden zur Laufzeit über `lib3d.LoadOBJ(path)` geladen (siehe
`docs/library.md`, Abschnitt 4.3). Beispiel: `apps/rocket/rocket.obj`
(Wavefront-OBJ, Y-up, Basis bei y=0, Spitze bei y=10, mit Drahtgitter-Linien).

Das OBJ liegt neben `main.go` im App-Verzeichnis und wird **relativ zum
Arbeitsverzeichnis** gesucht. Damit es gefunden wird:

- **Dev:** `make run-<name>` wechselt nach `apps/<name>`
  (`cd apps/<name> && go run .`), sodass `rocket.obj` direkt daneben liegt.
- **Build:** `make` kopiert jede `apps/<name>/*.obj` mit nach `bin/`, damit das
  Programm die Datei neben dem Binary findet (z. B. `cd bin && ./rocket`).

Diese Rakete wurde **einmalig** mit einem Wegwerf-Werkzeug erzeugt, das OBJ
selbst schreibt und **nicht** Teil der Bibliothek ist:

```bash
go run ./tmp/gen_rocket          # schreibt apps/rocket/rocket.obj
```

Der Ordner `tmp/` (mit `tmp/gen_rocket/main.go`) dient nur diesem Zweck und kann
gelöscht werden; `apps/rocket/rocket.obj` bleibt erhalten. Solange `tmp/`
existiert, wird es von `go build ./...` / `go test ./...` mitkompiliert.

### Abhängigkeiten

Die verwendeten Module stehen in `go.mod` / `go.sum`. Ein neuer Import, der
nicht im Modul-Cache liegt, erfordert `go mod tidy` (und damit i. d. R.
Netzwerkzugriff). Bitte vor dem Hinzufügen von Abhängigkeiten Rücksprache
halten.
