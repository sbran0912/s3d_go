# ====================================================================
# Makefile fuer simple3d (Go)
#
#   make             Alle Apps bauen (Alias: make build / make apps)
#   make build       Alle Apps nach bin/ bauen
#   make bin/<name>  Einzelne App bauen (z. B. make bin/ray)
#   make run APP=x   App x starten (x aus: $(APP_NAMES))
#   make run-x       Kurzform zum Starten (z. B. make run-ray)
#   make vet         go vet ./...
#   make test        Unit-Tests ausfuehren (go test ./...)
#   make clean       Build-Artefakte loeschen
#
# Verzeichnisse:
#   lib3d/     Bibliothek (Paket lib3d, inkl. Body/Faces/Raycasting)
#   apps/      Anwendungen, je App ein Unterordner (inkl. Mesh-Assets *.obj)
#   bin/       Fertige Programme (inkl. kopierter Mesh-Assets)
# ====================================================================

GO  ?= go
BIN := bin

APP_NAMES := beispiel vorlage instanzen ray boids rocket vehic
APP_BINS  := $(addprefix $(BIN)/,$(APP_NAMES))

.PHONY: all build apps vet test clean run $(APP_BINS) $(addprefix run-,$(APP_NAMES))

all: build

build: $(APP_BINS)

apps: build

# Immer neu bauen (go build cached inkrementell). Externe Mesh-Assets der App
# (apps/<name>/*.obj) werden mit nach bin/ kopiert, damit das Programm die
# Datei neben dem Binary findet.
$(APP_BINS):
	@mkdir -p $(BIN)
	$(GO) build -o $@ ./apps/$(notdir $@)
	@for a in apps/$(notdir $@)/*.obj; do \
		if [ -e "$$a" ]; then cp "$$a" $(BIN)/; fi; \
	done

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

# make run APP=<name>
run:
	@test -n "$(APP)" || { \
		echo "Fehler: APP fehlt."; \
		echo "Aufruf: make run APP=<name>   (verfuegbar: $(APP_NAMES))"; \
		exit 2; \
	}
	cd apps/$(APP) && $(GO) run .

# make run-<name>  (Kurzform). Ins App-Verzeichnis wechseln, damit das Programm
# dort liegende Assets (z. B. rocket.obj) relativ findet.
$(addprefix run-,$(APP_NAMES)): run-%:
	cd apps/$* && $(GO) run .

clean:
	rm -rf $(BIN)
