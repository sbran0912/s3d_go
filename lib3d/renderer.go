package lib3d

import (
	"errors"
	"math"
	"runtime"
	"sort"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// =================================================================
// OEFFENTLICHE API - Setup, Zustand, 3D-Primitive
//
// Koordinatensystem: +Y zeigt nach oben, Kamera blickt in +Z.
// View/Projection laufen im Vertex-Shader; die Modell-Matrix wird beim
// Sammeln in die Vertices gebacken und pro Zeichenaufruf uebergeben.
// =================================================================

// DrawStyle: 0 = stroke | 1 = fill | 2 = beide.
type DrawStyle int

const (
	StyleStroke DrawStyle = iota
	StyleFill
	StyleBoth
)

// EffectMode ist der aktive Shader-Effekt.
type EffectMode int

const (
	EffectFlat EffectMode = iota
	EffectGradient
	EffectPulse
)

// Key ist eine Auswahl haeufiger Tasten. Die Werte entsprechen exakt den
// GLFW_KEY_*-Codes, daher koennen die Polling-Funktionen auch direkt mit
// einem GLFW-Code aufgerufen werden (siehe IsKeyDown/IsKeyPressed).
type Key int32

const (
	KeySpace        Key = 32
	KeyApostrophe   Key = 39
	KeyComma        Key = 44
	KeyMinus        Key = 45
	KeyPeriod       Key = 46
	KeySlash        Key = 47
	Key0            Key = 48
	Key1            Key = 49
	Key2            Key = 50
	Key3            Key = 51
	Key4            Key = 52
	Key5            Key = 53
	Key6            Key = 54
	Key7            Key = 55
	Key8            Key = 56
	Key9            Key = 57
	KeySemicolon    Key = 59
	KeyEqual        Key = 61
	KeyA            Key = 65
	KeyB            Key = 66
	KeyC            Key = 67
	KeyD            Key = 68
	KeyE            Key = 69
	KeyF            Key = 70
	KeyG            Key = 71
	KeyH            Key = 72
	KeyI            Key = 73
	KeyJ            Key = 74
	KeyK            Key = 75
	KeyL            Key = 76
	KeyM            Key = 77
	KeyN            Key = 78
	KeyO            Key = 79
	KeyP            Key = 80
	KeyQ            Key = 81
	KeyR            Key = 82
	KeyS            Key = 83
	KeyT            Key = 84
	KeyU            Key = 85
	KeyV            Key = 86
	KeyW            Key = 87
	KeyX            Key = 88
	KeyY            Key = 89
	KeyZ            Key = 90
	KeyLeftBracket  Key = 91
	KeyBackslash    Key = 92
	KeyRightBracket Key = 93
	KeyGraveAccent  Key = 96
	KeyEscape       Key = 256
	KeyEnter        Key = 257
	KeyTab          Key = 258
	KeyBackspace    Key = 259
	KeyInsert       Key = 260
	KeyDelete       Key = 261
	KeyRight        Key = 262
	KeyLeft         Key = 263
	KeyDown         Key = 264
	KeyUp           Key = 265
	KeyPageUp       Key = 266
	KeyPageDown     Key = 267
	KeyHome         Key = 268
	KeyEnd          Key = 269
	KeyCapsLock     Key = 280
	KeyF1           Key = 290
	KeyF2           Key = 291
	KeyF3           Key = 292
	KeyF4           Key = 293
	KeyF5           Key = 294
	KeyF6           Key = 295
	KeyF7           Key = 296
	KeyF8           Key = 297
	KeyF9           Key = 298
	KeyF10          Key = 299
	KeyF11          Key = 300
	KeyF12          Key = 301
	KeyLeftShift    Key = 340
	KeyLeftControl  Key = 341
	KeyLeftAlt      Key = 342
	KeyLeftSuper    Key = 343
	KeyRightShift   Key = 344
	KeyRightControl Key = 345
	KeyRightAlt     Key = 346
	KeyRightSuper   Key = 347
)

// Instance beschreibt eine Instanz fuer das GPU-Instancing: eine Modell-Matrix
// (row-major, wie Matrix4x4) und eine Farbe.
type Instance struct {
	Model Matrix4x4
	Color ColorState
}

// =================================================================
// TYPEN & INTERNER ZUSTAND
// =================================================================

// DrawState ist der aktuelle Zeichenzustand (kein Stack).
type DrawState struct {
	fill   ColorState
	stroke ColorState
	lineW  float64
	effect EffectMode
	grad2  ColorState
}

// DrawCmd ist ein gesammelter Zeichenbefehl fuer eine Immediate-Primitive.
type DrawCmd struct {
	mode      uint32 // GL-Modus (GL_POINTS/LINES/TRIANGLES/LINE_LOOP)
	first     int    // Start-Index in batchVerts (Vertex-Nummer)
	count     int    // Anzahl Vertices
	col       ColorState
	effect    EffectMode
	grad2     ColorState
	pointSize float64
	lineW     float64
	center    [3]float64
	radius    float64
	view      [16]float32
	proj      [16]float32
	lit       int32 // 1 = gefuellte Flaeche -> beleuchten
}

// InstCmd ist ein gesammelter GPU-Instancing-Zeichenbefehl. first/count zeigen
// auf die Basis-Geometrie (instGeom), instFirst/instCount auf die Instanzen
// (instData).
type InstCmd struct {
	mode      uint32
	first     int
	count     int
	instFirst int
	instCount int
	effect    EffectMode
	grad2     ColorState
	pointSize float64
	lineW     float64
	center    [3]float64
	radius    float64
	view      [16]float32
	proj      [16]float32
	lit       int32
}

func identity16() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// Fenster / GL
var (
	window  *glfw.Window
	prog    uint32
	vao     uint32
	vbo     uint32
	vboGeom uint32 // Basis-Geometrie der Instanzen
	vboInst uint32 // Instanz-Daten (Modell-Matrix + Farbe)

	frameW, frameH int
)

// Shader-Locations
var (
	locPos        int32
	locColorAttr  int32
	locInstModel  int32 // belegt 4 aufeinanderfolgende Locations
	locInstColor  int32
	locInstanced  int32
	locView       int32
	locProjection int32
	locPointSize  int32
	locMode       int32
	locColor      int32
	locColor2     int32
	locTime       int32
	locCenter     int32
	locRadius     int32
	locFogNear    int32
	locFogFar     int32
	locFogColor   int32
	locLighted    int32
	locLightDir   int32
)

// Animation
var (
	looping   = true
	startTime float64
)

// Vollbild-Umschaltung (F11 / SetFullscreen).
var (
	fullscreen             bool
	toggleFullscreenWanted bool // im Loop anwenden (Re-Entrancy vermeiden)
	windowedPos            [2]int
	windowedSize           [2]int
)

// Maus
var (
	mouseX      float64
	mouseY      float64
	mouseStatus = 0
)

// Tastatur: Dauerdruck (keyDown) und einmalig konsumierte Druckkante
// (keyPressed) je Taste. Index ist der GLFW-key-Code.
const kMaxKeys = 512

var (
	keyDown    [kMaxKeys]bool
	keyPressed [kMaxKeys]bool
)

// Aktueller Zeichenzustand.
var state = DrawState{
	fill:   ColorState{1, 1, 1, 1},
	stroke: ColorState{0, 0, 0, 1},
	lineW:  1,
	effect: EffectFlat,
	grad2:  ColorState{0, 0, 0, 1},
}

// Batching: batchVerts ist interleaved (x,y,z,r,g,b,a = 7 floats/Vertex).
var (
	batchVerts []float32
	batchCmds  []DrawCmd
	batchCap   int // aktuell allozierte Byte-Groesse des VBO
)

// Optionale Sortierung/Umpacken der Batch (siehe SetBatchSort).
var batchSort = false

// Instancing: eigene Puffer fuer Basis-Geometrie und Instanz-Daten.
var (
	instGeom []float32 // interleaved xyz+rgba (7 floats/Vertex)
	instData []float32 // 16 (Modell, column-major) + 4 (Farbe) = 20 floats/Instanz
	instCmds []InstCmd
	geomCap  int
	instCap  int
)

// Zuletzt gesetzte Uniform-Werte (fuer den Batch-Snapshot pro Befehl).
var (
	viewUniform  = identity16()
	projUniform  = identity16()
	pointSizeVal = 4.0
	gradCenter   = [3]float64{0, 0, 0}
	gradRadius   = 1.0
)

// =================================================================
// HILFSFUNKTIONEN (intern)
// =================================================================

// flattenMatrix wandelt eine row-major Matrix4x4 in ein column-major
// Float32-Array fuer OpenGL.
func flattenMatrix(m Matrix4x4) [16]float32 {
	return [16]float32{
		float32(m[0][0]), float32(m[1][0]), float32(m[2][0]), float32(m[3][0]),
		float32(m[0][1]), float32(m[1][1]), float32(m[2][1]), float32(m[3][1]),
		float32(m[0][2]), float32(m[1][2]), float32(m[2][2]), float32(m[3][2]),
		float32(m[0][3]), float32(m[1][3]), float32(m[2][3]), float32(m[3][3]),
	}
}

func effectToMode(effect EffectMode) int32 {
	switch effect {
	case EffectGradient:
		return 1
	case EffectPulse:
		return 2
	default:
		return 0
	}
}

func arrEq(a, b [16]float32) bool {
	for i := 0; i < 16; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// canMerge prueft, ob zwei aufeinanderfolgende Befehle identischen Zustand
// haben und zu einem DrawArrays-Aufruf zusammengefasst werden koennen.
// LINE_LOOP wird nie zusammengefasst, weil ein Loop sich selbst schliesst.
func canMerge(a, b DrawCmd) bool {
	return a.mode == b.mode &&
		a.mode != gl.LINE_LOOP &&
		ColEq(a.col, b.col) &&
		a.effect == b.effect &&
		ColEq(a.grad2, b.grad2) &&
		a.pointSize == b.pointSize &&
		a.lineW == b.lineW &&
		a.center[0] == b.center[0] &&
		a.center[1] == b.center[1] &&
		a.center[2] == b.center[2] &&
		a.radius == b.radius &&
		arrEq(a.view, b.view) &&
		arrEq(a.proj, b.proj) &&
		a.lit == b.lit
}

// cmdKeyLess ist eine Sortier-Ordnung ueber den (fuer das Merging relevanten)
// Zustand eines Kommandos. view/proj sind pro Frame konstant und werden daher
// nicht verglichen.
func cmdKeyLess(a, b DrawCmd) bool {
	if a.mode != b.mode {
		return a.mode < b.mode
	}
	if a.lit != b.lit {
		return a.lit < b.lit
	}
	if a.effect != b.effect {
		return a.effect < b.effect
	}
	if a.lineW != b.lineW {
		return a.lineW < b.lineW
	}
	if a.pointSize != b.pointSize {
		return a.pointSize < b.pointSize
	}
	if a.radius != b.radius {
		return a.radius < b.radius
	}
	if a.center[0] != b.center[0] {
		return a.center[0] < b.center[0]
	}
	if a.center[1] != b.center[1] {
		return a.center[1] < b.center[1]
	}
	if a.center[2] != b.center[2] {
		return a.center[2] < b.center[2]
	}
	if a.col.R != b.col.R {
		return a.col.R < b.col.R
	}
	if a.col.G != b.col.G {
		return a.col.G < b.col.G
	}
	if a.col.B != b.col.B {
		return a.col.B < b.col.B
	}
	if a.col.A != b.col.A {
		return a.col.A < b.col.A
	}
	if a.grad2.R != b.grad2.R {
		return a.grad2.R < b.grad2.R
	}
	if a.grad2.G != b.grad2.G {
		return a.grad2.G < b.grad2.G
	}
	if a.grad2.B != b.grad2.B {
		return a.grad2.B < b.grad2.B
	}
	if a.grad2.A != b.grad2.A {
		return a.grad2.A < b.grad2.A
	}
	return false
}

// mergeBlock haengt den Vertex-Block von src an dst an und legt (bei gleichem
// Zustand wie der letzte Befehl) die Kommandos zusammen.
func mergeBlock(dstData *[]float32, dstCmds *[]DrawCmd, src DrawCmd, srcData []float32) {
	c := src
	begin := c.first * 7
	end := begin + c.count*7
	c.first = len(*dstData) / 7
	*dstData = append(*dstData, srcData[begin:end]...)
	if n := len(*dstCmds); n > 0 && canMerge((*dstCmds)[n-1], c) {
		(*dstCmds)[n-1].count += c.count
	} else {
		*dstCmds = append(*dstCmds, c)
	}
}

// =================================================================
// BATCHING - SAMMELN & ZEICHNEN (Kern des Renderers)
// =================================================================

// submit sammelt eine Immediate-Primitive im Frame-Batch. Es wird noch nichts
// an die GPU geschickt; das passiert erst beim FlushBatch(). Die Modell-Matrix
// wird hier in die Vertices gebacken und die Farbe pro Vertex abgelegt.
func submit(mode uint32, verts []float32, useStroke bool, model Matrix4x4) {
	col := state.fill
	lit := int32(1) // gefuellte Flaechen werden beleuchtet
	if useStroke {
		col = state.stroke
		lit = 0
	}
	flat := state.effect == EffectFlat

	// Flache Primitive tragen ihre Farbe pro Vertex (uColor bleibt weiss);
	// Gradient/Pulse nutzen uColor (Vertex-Farbe bleibt weiss).
	vertexCol := ColorState{1, 1, 1, 1}
	uniformCol := ColorState{1, 1, 1, 1}
	if flat {
		vertexCol = col
	} else {
		uniformCol = col
	}

	cmd := DrawCmd{
		mode:      mode,
		first:     len(batchVerts) / 7,
		count:     len(verts) / 3,
		col:       uniformCol,
		effect:    state.effect,
		grad2:     state.grad2,
		pointSize: pointSizeVal,
		lineW:     state.lineW,
		center:    gradCenter,
		radius:    gradRadius,
		view:      viewUniform,
		proj:      projUniform,
		lit:       lit,
	}

	// Vertices mit der Modell-Matrix backen und mit Farbe interleaven.
	m := model
	n := len(verts) / 3
	for i := 0; i < n; i++ {
		x := float64(verts[i*3+0])
		y := float64(verts[i*3+1])
		z := float64(verts[i*3+2])
		batchVerts = append(batchVerts,
			float32(m[0][0]*x+m[0][1]*y+m[0][2]*z+m[0][3]),
			float32(m[1][0]*x+m[1][1]*y+m[1][2]*z+m[1][3]),
			float32(m[2][0]*x+m[2][1]*y+m[2][2]*z+m[2][3]),
			float32(vertexCol.R), float32(vertexCol.G), float32(vertexCol.B), float32(vertexCol.A),
		)
	}

	// Aufeinanderfolgende Befehle mit identischem Zustand zusammenfassen.
	if n := len(batchCmds); n > 0 && canMerge(batchCmds[n-1], cmd) {
		batchCmds[n-1].count += cmd.count
		return
	}
	batchCmds = append(batchCmds, cmd)
}

// submitTriangles sammelt gefuellte Dreiecke (Fill-Farbe, beleuchtet).
func submitTriangles(verts []float32, model Matrix4x4) {
	submit(gl.TRIANGLES, verts, false, model)
}

// submitLines sammelt Linien (Stroke-Farbe, unbelichtet).
func submitLines(verts []float32, model Matrix4x4) {
	submit(gl.LINES, verts, true, model)
}

// submitInstanced sammelt einen GPU-Instancing-Zeichenauftrag: lokale
// Mesh-Dreiecke/-Linien plus eine Liste von Instanzen (Modell-Matrix + Farbe).
func submitInstanced(triangles, lines []float32, style DrawStyle, instances []Instance) {
	if len(instances) == 0 {
		return
	}

	// Instanz-Daten: Modell-Matrix (column-major) + Farbe.
	instFirst := len(instData) / 20
	for _, in := range instances {
		m := in.Model
		instData = append(instData,
			float32(m[0][0]), float32(m[1][0]), float32(m[2][0]), float32(m[3][0]),
			float32(m[0][1]), float32(m[1][1]), float32(m[2][1]), float32(m[3][1]),
			float32(m[0][2]), float32(m[1][2]), float32(m[2][2]), float32(m[3][2]),
			float32(m[0][3]), float32(m[1][3]), float32(m[2][3]), float32(m[3][3]),
			float32(in.Color.R), float32(in.Color.G), float32(in.Color.B), float32(in.Color.A),
		)
	}
	instCount := len(instances)

	// Basis-Geometrie anhaengen (Farbe weiss; pro Instanz ueberschrieben).
	appendGeom := func(dst, src []float32) []float32 {
		for i := 0; i+2 < len(src); i += 3 {
			dst = append(dst, src[i], src[i+1], src[i+2], 1, 1, 1, 1)
		}
		return dst
	}

	if (style == StyleFill || style == StyleBoth) && len(triangles) > 0 {
		c := InstCmd{
			mode:      gl.TRIANGLES,
			first:     len(instGeom) / 7,
			count:     len(triangles) / 3,
			effect:    state.effect,
			grad2:     state.grad2,
			pointSize: pointSizeVal,
			lineW:     state.lineW,
			center:    gradCenter,
			radius:    gradRadius,
			view:      viewUniform,
			proj:      projUniform,
			lit:       1,
		}
		instGeom = appendGeom(instGeom, triangles)
		c.instFirst = instFirst
		c.instCount = instCount
		instCmds = append(instCmds, c)
	}
	if (style == StyleStroke || style == StyleBoth) && len(lines) > 0 {
		c := InstCmd{
			mode:      gl.LINES,
			first:     len(instGeom) / 7,
			count:     len(lines) / 3,
			effect:    state.effect,
			grad2:     state.grad2,
			pointSize: pointSizeVal,
			lineW:     state.lineW,
			center:    gradCenter,
			radius:    gradRadius,
			view:      viewUniform,
			proj:      projUniform,
			lit:       0,
		}
		instGeom = appendGeom(instGeom, lines)
		c.instFirst = instFirst
		c.instCount = instCount
		instCmds = append(instCmds, c)
	}
}

// flushInstanced zeichnet alle gesammelten Instancing-Auftraege. Basis-
// Geometrie und Instanz-Daten liegen in eigenen, wiederverwendeten VBOs.
func flushInstanced() {
	if len(instCmds) == 0 {
		return
	}

	gl.BindVertexArray(vao)

	// Basis-Geometrie hochladen.
	gl.BindBuffer(gl.ARRAY_BUFFER, vboGeom)
	geomBytes := len(instGeom) * 4
	if geomBytes > geomCap {
		gl.BufferData(gl.ARRAY_BUFFER, geomBytes, unsafe.Pointer(&instGeom[0]), gl.DYNAMIC_DRAW)
		geomCap = geomBytes
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, geomBytes, unsafe.Pointer(&instGeom[0]))
	}
	gStride := int32(7 * 4)
	gl.EnableVertexAttribArray(uint32(locPos))
	gl.EnableVertexAttribArray(uint32(locColorAttr))
	gl.VertexAttribPointer(uint32(locPos), 3, gl.FLOAT, false, gStride, nil)
	gl.VertexAttribPointer(uint32(locColorAttr), 4, gl.FLOAT, false, gStride, unsafe.Add(nil, 3*4))
	gl.VertexAttribDivisor(uint32(locPos), 0)
	gl.VertexAttribDivisor(uint32(locColorAttr), 0)

	// Instanz-Daten hochladen (20 floats/Instanz).
	gl.BindBuffer(gl.ARRAY_BUFFER, vboInst)
	instBytes := len(instData) * 4
	if instBytes > instCap {
		gl.BufferData(gl.ARRAY_BUFFER, instBytes, unsafe.Pointer(&instData[0]), gl.DYNAMIC_DRAW)
		instCap = instBytes
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, instBytes, unsafe.Pointer(&instData[0]))
	}
	iStride := int32(20 * 4)
	for k := 0; k < 4; k++ {
		loc := uint32(locInstModel + int32(k))
		gl.EnableVertexAttribArray(loc)
		gl.VertexAttribPointer(loc, 4, gl.FLOAT, false, iStride, unsafe.Add(nil, k*4*4))
		gl.VertexAttribDivisor(loc, 1)
	}
	gl.EnableVertexAttribArray(uint32(locInstColor))
	gl.VertexAttribPointer(uint32(locInstColor), 4, gl.FLOAT, false, iStride, unsafe.Add(nil, 16*4))
	gl.VertexAttribDivisor(uint32(locInstColor), 1)

	gl.Uniform1i(locInstanced, 1)
	gl.Uniform4f(locColor, 1, 1, 1, 1)
	for _, c := range instCmds {
		// glDrawArraysInstanced kennt in GL 3.3 keinen baseInstance-Parameter.
		// Daher muessen die Instanz-Attribute pro Kommando auf den Start der
		// jeweiligen Instanzen (instFirst) zeigen - sonst liest jedes Kommando
		// ab Instanz 0 und mehrere Instancing-Draws vermischen sich.
		instOffset := c.instFirst * 20 * 4
		for k := 0; k < 4; k++ {
			loc := uint32(locInstModel + int32(k))
			gl.VertexAttribPointer(loc, 4, gl.FLOAT, false, iStride, unsafe.Add(nil, instOffset+k*4*4))
		}
		gl.VertexAttribPointer(uint32(locInstColor), 4, gl.FLOAT, false, iStride, unsafe.Add(nil, instOffset+16*4))

		gl.UniformMatrix4fv(locView, 1, false, &c.view[0])
		gl.UniformMatrix4fv(locProjection, 1, false, &c.proj[0])
		gl.Uniform1i(locMode, effectToMode(c.effect))
		gl.Uniform1i(locLighted, c.lit)
		gl.Uniform4f(locColor2, float32(c.grad2.R), float32(c.grad2.G), float32(c.grad2.B), float32(c.grad2.A))
		gl.Uniform1f(locPointSize, float32(c.pointSize))
		gl.Uniform3f(locCenter, float32(c.center[0]), float32(c.center[1]), float32(c.center[2]))
		gl.Uniform1f(locRadius, float32(c.radius))
		gl.LineWidth(float32(c.lineW))
		gl.DrawArraysInstanced(c.mode, int32(c.first), int32(c.count), int32(c.instCount))
	}
	gl.Uniform1i(locInstanced, 0)

	// Divisoren/Attribute zuruecksetzen, damit normale Draws sauber bleiben.
	for k := 0; k < 4; k++ {
		loc := uint32(locInstModel + int32(k))
		gl.VertexAttribDivisor(loc, 0)
		gl.DisableVertexAttribArray(loc)
	}
	gl.VertexAttribDivisor(uint32(locInstColor), 0)
	gl.DisableVertexAttribArray(uint32(locInstColor))

	instGeom = instGeom[:0]
	instData = instData[:0]
	instCmds = instCmds[:0]
}

// FlushBatch zeichnet alle gesammelten Primitives des Frames mit einem
// einzigen wiederverwendbaren VBO.
func FlushBatch() {
	if len(batchCmds) == 0 && len(instCmds) == 0 {
		return
	}

	CompactBatch()

	gl.Uniform1i(locInstanced, 0)

	if len(batchCmds) > 0 {
		gl.BindVertexArray(vao)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

		byteLen := len(batchVerts) * 4
		if byteLen > batchCap {
			gl.BufferData(gl.ARRAY_BUFFER, byteLen, unsafe.Pointer(&batchVerts[0]), gl.DYNAMIC_DRAW)
			batchCap = byteLen
		} else {
			gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteLen, unsafe.Pointer(&batchVerts[0]))
		}

		gl.EnableVertexAttribArray(uint32(locPos))
		gl.EnableVertexAttribArray(uint32(locColorAttr))
		stride := int32(7 * 4)
		gl.VertexAttribPointer(uint32(locPos), 3, gl.FLOAT, false, stride, nil)
		gl.VertexAttribPointer(uint32(locColorAttr), 4, gl.FLOAT, false, stride, unsafe.Add(nil, 3*4))

		for _, c := range batchCmds {
			gl.UniformMatrix4fv(locView, 1, false, &c.view[0])
			gl.UniformMatrix4fv(locProjection, 1, false, &c.proj[0])
			gl.Uniform1i(locMode, effectToMode(c.effect))
			gl.Uniform1i(locLighted, c.lit)
			gl.Uniform4f(locColor, float32(c.col.R), float32(c.col.G), float32(c.col.B), float32(c.col.A))
			gl.Uniform4f(locColor2, float32(c.grad2.R), float32(c.grad2.G), float32(c.grad2.B), float32(c.grad2.A))
			gl.Uniform1f(locPointSize, float32(c.pointSize))
			gl.Uniform3f(locCenter, float32(c.center[0]), float32(c.center[1]), float32(c.center[2]))
			gl.Uniform1f(locRadius, float32(c.radius))
			gl.LineWidth(float32(c.lineW))
			gl.DrawArrays(c.mode, int32(c.first), int32(c.count))
		}

		batchVerts = batchVerts[:0]
		batchCmds = batchCmds[:0]
	}

	flushInstanced()
}

// =================================================================
// MAUS / FENSTER (intern)
// =================================================================

func onCursorPos(_ *glfw.Window, x, y float64) {
	mouseX = x - float64(frameW)/2
	mouseY = -(y - float64(frameH)/2)
}

func onMouseButton(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
	if button != glfw.MouseButtonLeft {
		return
	}
	switch action {
	case glfw.Press:
		mouseStatus = 1
	case glfw.Release:
		mouseStatus = 2
	}
}

// onKey pflegt den Tastatur-Zustand (IsKeyDown/IsKeyPressed) und beendet das
// Programm sauber mit ESC.
func onKey(win *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
	k := int(key)
	if k >= 0 && k < kMaxKeys {
		switch action {
		case glfw.Press:
			keyDown[k] = true
			keyPressed[k] = true // Kante merken (GLFW_REPEAT zaehlt nicht)
		case glfw.Release:
			keyDown[k] = false
		}
	}
	if action == glfw.Press && key == glfw.KeyEscape {
		win.SetShouldClose(true)
	}
	// F11 schaltet den Vollbildmodus um. Der eigentliche Wechsel passiert erst
	// im Loop (nach PollEvents), damit SetMonitor nicht mitten im Callback
	// laeuft.
	if action == glfw.Press && key == glfw.KeyF11 {
		toggleFullscreenWanted = true
	}
}

func onFramebufferSize(_ *glfw.Window, width, height int) {
	frameW, frameH = width, height
	gl.Viewport(0, 0, int32(width), int32(height))
}

// =================================================================
// OEFFENTLICHE API - Setup & Matrizen
// =================================================================

// GetWidth liefert die Fenstergroesse in Pixeln.
func GetWidth() int { return frameW }

// GetHeight liefert die Fenstergroesse in Pixeln.
func GetHeight() int { return frameH }

// NoLoop stoppt die Animations-Schleife.
func NoLoop() { looping = false }

// IsMouseDown gibt true zurueck, solange die linke Maustaste gedrueckt ist.
func IsMouseDown() bool { return mouseStatus == 1 }

// IsMouseUp gibt einmalig true zurueck, wenn die Maustaste losgelassen wurde.
func IsMouseUp() bool {
	if mouseStatus == 2 {
		mouseStatus = 0
		return true
	}
	return false
}

// IsFullscreen gibt true zurueck, wenn das Fenster im Vollbildmodus ist.
func IsFullscreen() bool { return fullscreen }

// SetFullscreen schaltet zwischen Fenster- und Vollbildmodus um. Beide
// Richtungen erhalten den OpenGL-Kontext; Fenstergroesse (und damit
// Viewport/Aspect) werden automatisch aktualisiert. Ohne verfuegbaren Monitor
// bleibt das Fenster unveraendert (kein Vollbild).
func SetFullscreen(on bool) {
	if window == nil || on == fullscreen {
		return
	}
	if on {
		monitor := glfw.GetPrimaryMonitor()
		var mode *glfw.VidMode
		if monitor != nil {
			mode = monitor.GetVideoMode()
		}
		if monitor == nil || mode == nil {
			return // ohne Monitor kein Vollbild
		}
		windowedPos[0], windowedPos[1] = window.GetPos()
		windowedSize[0], windowedSize[1] = window.GetSize()
		window.SetMonitor(monitor, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
	} else {
		window.SetMonitor(nil, windowedPos[0], windowedPos[1], windowedSize[0], windowedSize[1], 0)
	}
	fullscreen = on

	// Framebuffer-Groesse sofort uebernehmen (der Callback feuert ggf. erst
	// beim naechsten Poll).
	frameW, frameH = window.GetFramebufferSize()
	gl.Viewport(0, 0, int32(frameW), int32(frameH))
}

// ToggleFullscreen schaltet zwischen Fenster- und Vollbildmodus um.
func ToggleFullscreen() { SetFullscreen(!fullscreen) }

// IsKeyDown gibt true zurueck, solange die Taste gedrueckt ist.
func IsKeyDown(key Key) bool {
	k := int(key)
	if k < 0 || k >= kMaxKeys {
		return false
	}
	return keyDown[k]
}

// IsKeyPressed gibt einmalig true zurueck, wenn die Taste seit der letzten
// Abfrage gedrueckt wurde (konsumierend wie IsMouseUp).
func IsKeyPressed(key Key) bool {
	k := int(key)
	if k < 0 || k >= kMaxKeys {
		return false
	}
	if keyPressed[k] {
		keyPressed[k] = false
		return true
	}
	return false
}

// SetProjection setzt die Projection-Matrix.
func SetProjection(m Matrix4x4) {
	projUniform = flattenMatrix(m)
	gl.UniformMatrix4fv(locProjection, 1, false, &projUniform[0])
}

// SetView setzt die View-Matrix (Kamera).
func SetView(m Matrix4x4) {
	viewUniform = flattenMatrix(m)
	gl.UniformMatrix4fv(locView, 1, false, &viewUniform[0])
}

// ApplyCamera setzt View + Projection (Aspect aus GetWidth/GetHeight).
func ApplyCamera(cam Camera) {
	SetProjection(cam.Projection(float64(GetWidth()) / float64(GetHeight())))
	SetView(cam.View())
}

// BeginScene wendet die Szene an: Hintergrund, Nebel, Kamera und
// Sonnenrichtung (in Kamerakoordinaten). Pro Frame aufzurufen.
func BeginScene(scene Scene) {
	Background(scene.Background)
	if scene.SceneFog.Enabled {
		c := scene.SceneFog.Color
		SetFog(scene.SceneFog.Near, scene.SceneFog.Far, c.R, c.G, c.B, c.A)
	}
	ApplyCamera(scene.Cam)
	camLight := scene.SunDir.TransformDir(scene.Cam.View())
	SetLightDirection(camLight.X, camLight.Y, camLight.Z)
}

// GetPendingDrawCalls liefert die Anzahl der im aktuellen Batch noch nicht
// gezeichneten Draw-Calls. Fuer Tests/Diagnose gedacht (kein GL-Zugriff).
func GetPendingDrawCalls() int { return len(batchCmds) }

// GetPendingInstancedDrawCalls liefert analog die Anzahl der gesammelten
// Instancing-Draw-Calls (fuer Tests/Diagnose, kein GL-Zugriff).
func GetPendingInstancedDrawCalls() int { return len(instCmds) }

// SetBatchSort aktiviert das Sortieren/Umpacken der Pending-Batch, damit auch
// nicht-benachbarte Primitive mit gleichem Zustand zusammengefasst werden.
// ACHTUNG: aendert die Zeichenreihenfolge -> nur fuer opake Geometrie geeignet.
func SetBatchSort(on bool) { batchSort = on }

// CompactBatch sortiert (falls aktiviert) und fasst die Pending-Batch zusammen.
func CompactBatch() {
	if !batchSort || len(batchCmds) < 2 {
		return
	}

	order := make([]int, len(batchCmds))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool {
		return cmdKeyLess(batchCmds[order[x]], batchCmds[order[y]])
	})

	newData := make([]float32, 0, len(batchVerts))
	var newCmds []DrawCmd
	for _, idx := range order {
		mergeBlock(&newData, &newCmds, batchCmds[idx], batchVerts)
	}
	batchVerts = newData
	batchCmds = newCmds
}

// SetGradientCenter setzt das Zentrum fuer den Gradient-Effekt.
func SetGradientCenter(cx, cy, cz, radius float64) {
	gradCenter = [3]float64{cx, cy, cz}
	gradRadius = radius
	gl.Uniform3f(locCenter, float32(cx), float32(cy), float32(cz))
	gl.Uniform1f(locRadius, float32(radius))
}

// SetFog aktiviert Tiefen-Nebel.
func SetFog(near, far, r, g, b, a float64) {
	gl.Uniform1f(locFogNear, float32(near))
	gl.Uniform1f(locFogFar, float32(far))
	gl.Uniform4f(locFogColor, float32(r), float32(g), float32(b), float32(a))
}

// SetLightDirection setzt die Lichtrichtung fuer das Flat-Shading in
// Kamerakoordinaten (zeigt vom Fragment ZUR Lichtquelle).
func SetLightDirection(x, y, z float64) {
	gl.Uniform3f(locLightDir, float32(x), float32(y), float32(z))
}

// defaultWindowSize liefert die Standard-Fenstergroesse im Fenstermodus:
// zentriert 1280x800, auf kleinen Monitoren automatisch begrenzt.
func defaultWindowSize() (int, int) {
	const wantW, wantH = 1280, 800
	w, h := wantW, wantH
	if monitor := glfw.GetPrimaryMonitor(); monitor != nil {
		if mode := monitor.GetVideoMode(); mode != nil {
			w = min(wantW, mode.Width-100)
			h = min(wantH, mode.Height-100)
		}
	}
	return max(w, 320), max(h, 240)
}

// Init initialisiert das Fenster im Fenstermodus (zentriert, 1280x800),
// den OpenGL-Kontext, Shader und Batch-Zustand. Per F11 (oder SetFullscreen)
// laesst sich zur Laufzeit auf Vollbild umschalten. Liefert einen Fehler,
// wenn das Fenster oder der GL-Kontext nicht erstellt werden kann.
func Init() error {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Samples, 4)

	// Fenstermodus als Standard (kein exklusives Vollbild). Der Vollbild-
	// Wechsel erfolgt zur Laufzeit ueber SetMonitor und erhaelt den Kontext.
	winW, winH := defaultWindowSize()

	win, err := glfw.CreateWindow(winW, winH, "3D Renderer", nil, nil)
	if err != nil {
		glfw.Terminate()
		return err
	}
	window = win
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	// Fenster auf dem primaeren Monitor zentrieren (sofern moeglich).
	if monitor := glfw.GetPrimaryMonitor(); monitor != nil {
		if mode := monitor.GetVideoMode(); mode != nil {
			window.SetPos((mode.Width-winW)/2, (mode.Height-winH)/2)
		}
	}

	if err := gl.Init(); err != nil {
		glfw.Terminate()
		return err
	}

	program, err := createProgram(vertSrc, fragSrc)
	if err != nil {
		glfw.Terminate()
		return err
	}
	prog = program
	gl.UseProgram(prog)

	locPos = gl.GetAttribLocation(prog, gl.Str("aPos\x00"))
	locColorAttr = gl.GetAttribLocation(prog, gl.Str("aColor\x00"))
	locInstModel = gl.GetAttribLocation(prog, gl.Str("aInstModel\x00"))
	locInstColor = gl.GetAttribLocation(prog, gl.Str("aInstColor\x00"))
	locInstanced = gl.GetUniformLocation(prog, gl.Str("uInstanced\x00"))
	locView = gl.GetUniformLocation(prog, gl.Str("uView\x00"))
	locProjection = gl.GetUniformLocation(prog, gl.Str("uProjection\x00"))
	locPointSize = gl.GetUniformLocation(prog, gl.Str("uPointSize\x00"))
	locMode = gl.GetUniformLocation(prog, gl.Str("uMode\x00"))
	locColor = gl.GetUniformLocation(prog, gl.Str("uColor\x00"))
	locColor2 = gl.GetUniformLocation(prog, gl.Str("uColor2\x00"))
	locTime = gl.GetUniformLocation(prog, gl.Str("uTime\x00"))
	locCenter = gl.GetUniformLocation(prog, gl.Str("uShapeCenter\x00"))
	locRadius = gl.GetUniformLocation(prog, gl.Str("uShapeRadius\x00"))
	locFogNear = gl.GetUniformLocation(prog, gl.Str("uFogNear\x00"))
	locFogFar = gl.GetUniformLocation(prog, gl.Str("uFogFar\x00"))
	locFogColor = gl.GetUniformLocation(prog, gl.Str("uFogColor\x00"))
	locLighted = gl.GetUniformLocation(prog, gl.Str("uLighted\x00"))
	locLightDir = gl.GetUniformLocation(prog, gl.Str("uLightDir\x00"))

	// VAO/VBO anlegen (OpenGL 3.3 Core benoetigt ein VAO).
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.GenBuffers(1, &vboGeom)
	gl.GenBuffers(1, &vboInst)

	gl.Uniform1f(locPointSize, 4.0)
	gl.Uniform3f(locCenter, 0, 0, 0)
	gl.Uniform1f(locRadius, 1)
	gl.Uniform1i(locLighted, 0)
	gl.Uniform1i(locInstanced, 0)
	gl.Uniform3f(locLightDir, 0, 0, -1) // Headlight: Kamera blickt +Z

	ident := identity16()
	gl.UniformMatrix4fv(locProjection, 1, false, &ident[0])
	gl.UniformMatrix4fv(locView, 1, false, &ident[0])

	// Fog-Defaults (kein Nebel)
	gl.Uniform1f(locFogNear, 0)
	gl.Uniform1f(locFogFar, 1)
	gl.Uniform4f(locFogColor, 0, 0, 0, 0)

	// Depth-Testing + Alpha-Blending.
	// LEQUAL (statt LESS): koplanare Kanten bleiben ueber ihren eigenen
	// Flaechen sichtbar.
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.PROGRAM_POINT_SIZE)

	// Groesse/Viewport aus dem tatsaechlichen Framebuffer uebernehmen.
	frameW, frameH = window.GetFramebufferSize()
	if frameW <= 0 || frameH <= 0 {
		glfw.Terminate()
		return errors.New("ungueltige Framebuffer-Groesse")
	}
	gl.Viewport(0, 0, int32(frameW), int32(frameH))
	startTime = glfw.GetTime()

	// Batch-Zustand initialisieren.
	state = DrawState{
		fill:   ColorState{1, 1, 1, 1},
		stroke: ColorState{0, 0, 0, 1},
		lineW:  1,
		effect: EffectFlat,
		grad2:  ColorState{0, 0, 0, 1},
	}
	viewUniform = identity16()
	projUniform = identity16()
	batchSort = false
	geomCap = 0
	instCap = 0
	instGeom = instGeom[:0]
	instData = instData[:0]
	instCmds = instCmds[:0]
	pointSizeVal = 4.0
	gradCenter = [3]float64{0, 0, 0}
	gradRadius = 1
	batchCap = 0
	batchVerts = batchVerts[:0]
	batchCmds = batchCmds[:0]

	// Tastatur-Zustand zuruecksetzen.
	keyDown = [kMaxKeys]bool{}
	keyPressed = [kMaxKeys]bool{}

	// Vollbild-Zustand zuruecksetzen.
	fullscreen = false
	toggleFullscreenWanted = false

	// Eingabe-Callbacks.
	window.SetCursorPosCallback(onCursorPos)
	window.SetMouseButtonCallback(onMouseButton)
	window.SetKeyCallback(onKey)
	window.SetFramebufferSizeCallback(onFramebufferSize)

	return nil
}

// StartAnimation startet die Animations-Schleife. fnDraw wird jedes Frame
// aufgerufen; danach zeichnet die Bibliothek automatisch den Batch.
func StartAnimation(fnDraw func()) {
	looping = true
	for looping && !window.ShouldClose() {
		t := glfw.GetTime() - startTime
		gl.Uniform1f(locTime, float32(t))

		fnDraw()

		FlushBatch()

		window.SwapBuffers()
		glfw.PollEvents()

		// F11-Umschaltung (nach PollEvents anwenden, siehe onKey).
		if toggleFullscreenWanted {
			toggleFullscreenWanted = false
			ToggleFullscreen()
		}
	}
	glfw.Terminate()
}

// =================================================================
// OEFFENTLICHE API - Zeichenzustand
// =================================================================

// FillColor setzt die Fuellfarbe.
func FillColor(color ColorState) { state.fill = color }

// FillColorInput setzt die Fuellfarbe aus einem ColorInput (nur wenn gesetzt).
func FillColorInput(color ColorInput) { FillColor(color.Color) }

// StrokeColor setzt die Linienfarbe.
func StrokeColor(color ColorState) { state.stroke = color }

// StrokeColorInput setzt die Linienfarbe aus einem ColorInput.
func StrokeColorInput(color ColorInput) { StrokeColor(color.Color) }

// StrokeWidth setzt die Linienstaerke in Pixeln.
func StrokeWidth(w float64) { state.lineW = w }

// SetEffect waehlt den aktiven Shader-Effekt.
func SetEffect(effect EffectMode) { state.effect = effect }

// SetGradient setzt die zweite Farbe fuer den Gradient-Effekt.
func SetGradient(color ColorState) { state.grad2 = color }

// SetGradientInput setzt die zweite Farbe fuer den Gradient-Effekt.
func SetGradientInput(color ColorInput) { SetGradient(color.Color) }

// Background loescht den gesamten Canvas inkl. Tiefenpuffer.
func Background(color ColorState) {
	FlushBatch()
	gl.ClearColor(float32(color.R), float32(color.G), float32(color.B), float32(color.A))
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// BackgroundInput loescht den Canvas inkl. Tiefenpuffer (ColorInput).
func BackgroundInput(color ColorInput) { Background(color.Color) }

// =================================================================
// OEFFENTLICHE API - 3D-Primitive
//
// Jeder Aufruf traegt seine Modell-Matrix explizit als letztes Argument
// (Default: Identity). Damit gibt es keinen globalen Modell-Zustand mehr.
// =================================================================

// modelOr liefert die optionale Modell-Matrix oder die Identitaet.
func modelOr(m []Matrix4x4) Matrix4x4 {
	if len(m) > 0 {
		return m[0]
	}
	return IdentityMatrix()
}

// PointSize setzt die Punktgroesse in Pixeln.
func PointSize(px float64) {
	pointSizeVal = px
	gl.Uniform1f(locPointSize, float32(px))
}

// Point zeichnet einen 3D-Punkt bei (x,y,z).
func Point(x, y, z float64, model ...Matrix4x4) {
	submit(gl.POINTS, []float32{float32(x), float32(y), float32(z)}, true, modelOr(model))
}

// Line zeichnet eine 3D-Linie von (x1,y1,z1) nach (x2,y2,z2).
func Line(x1, y1, z1, x2, y2, z2 float64, model ...Matrix4x4) {
	submit(gl.LINES, []float32{
		float32(x1), float32(y1), float32(z1),
		float32(x2), float32(y2), float32(z2),
	}, true, modelOr(model))
}

// Triangle zeichnet ein 3D-Dreieck (style: 0=stroke | 1=fill | 2=beide).
func Triangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float64, style DrawStyle, model ...Matrix4x4) {
	pts := []float32{
		float32(x1), float32(y1), float32(z1),
		float32(x2), float32(y2), float32(z2),
		float32(x3), float32(y3), float32(z3),
	}
	m := modelOr(model)

	if style == StyleFill || style == StyleBoth {
		submit(gl.TRIANGLES, pts, false, m)
	}
	if style == StyleStroke || style == StyleBoth {
		submit(gl.LINES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2),
			float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x3), float32(y3), float32(z3), float32(x1), float32(y1), float32(z1),
		}, true, m)
	}
}

// Shape zeichnet ein 3D-Viereck mit 4 beliebigen Punkten.
func Shape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float64, style DrawStyle, model ...Matrix4x4) {
	m := modelOr(model)
	if style == StyleFill || style == StyleBoth {
		submit(gl.TRIANGLES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x1), float32(y1), float32(z1), float32(x3), float32(y3), float32(z3), float32(x4), float32(y4), float32(z4),
		}, false, m)
	}
	if style == StyleStroke || style == StyleBoth {
		submit(gl.LINES, []float32{
			float32(x1), float32(y1), float32(z1), float32(x2), float32(y2), float32(z2),
			float32(x2), float32(y2), float32(z2), float32(x3), float32(y3), float32(z3),
			float32(x3), float32(y3), float32(z3), float32(x4), float32(y4), float32(z4),
			float32(x4), float32(y4), float32(z4), float32(x1), float32(y1), float32(z1),
		}, true, m)
	}
}

// Rect zeichnet ein achsenparalleles Rechteck in der XY-Ebene bei z.
func Rect(x, y, w, h float64, style DrawStyle, z float64, model ...Matrix4x4) {
	hw, hh := w/2, h/2
	Shape(x-hw, y-hh, z, x+hw, y-hh, z, x+hw, y+hh, z, x-hw, y+hh, z, style, modelOr(model))
}

// Circle zeichnet einen 3D-Kreis (Ring) in der XY-Ebene bei z.
func Circle(x, y, z, radius float64, style DrawStyle, segments int, model ...Matrix4x4) {
	tau := 2 * math.Pi
	m := modelOr(model)

	if style == StyleFill || style == StyleBoth {
		fillVerts := make([]float32, 0, segments*9)
		for i := 0; i < segments; i++ {
			a0 := (float64(i) / float64(segments)) * tau
			a1 := (float64(i+1) / float64(segments)) * tau
			fillVerts = append(fillVerts,
				float32(x), float32(y), float32(z),
				float32(x+math.Cos(a0)*radius), float32(y+math.Sin(a0)*radius), float32(z),
				float32(x+math.Cos(a1)*radius), float32(y+math.Sin(a1)*radius), float32(z),
			)
		}
		submit(gl.TRIANGLES, fillVerts, false, m)
	}

	if style == StyleStroke || style == StyleBoth {
		strokeVerts := make([]float32, 0, segments*3)
		for i := 0; i < segments; i++ {
			a := (float64(i) / float64(segments)) * tau
			strokeVerts = append(strokeVerts,
				float32(x+math.Cos(a)*radius),
				float32(y+math.Sin(a)*radius),
				float32(z),
			)
		}
		submit(gl.LINE_LOOP, strokeVerts, true, m)
	}
}

// Polygon zeichnet ein beliebiges 3D-Polygon als flaches Array
// [x0,y0,z0, x1,y1,z1, ...] (style: 0=stroke | 1=fill | 2=beide).
// Hinweis: fill ist korrekt nur fuer konvexe Polygone (Fan-Triangulation).
func Polygon(pts []float64, style DrawStyle, model ...Matrix4x4) {
	if len(pts) < 4 {
		return
	}
	n := len(pts) / 3
	m := modelOr(model)

	if style == StyleFill || style == StyleBoth {
		fillVerts := make([]float32, 0)
		for i := 1; i < n-1; i++ {
			fillVerts = append(fillVerts,
				float32(pts[0]), float32(pts[1]), float32(pts[2]),
				float32(pts[i*3]), float32(pts[i*3+1]), float32(pts[i*3+2]),
				float32(pts[i*3+3]), float32(pts[i*3+4]), float32(pts[i*3+5]),
			)
		}
		submit(gl.TRIANGLES, fillVerts, false, m)
	}
	if style == StyleStroke || style == StyleBoth {
		strokeVerts := make([]float32, len(pts))
		for i, v := range pts {
			strokeVerts[i] = float32(v)
		}
		submit(gl.LINE_LOOP, strokeVerts, true, m)
	}
}

// =================================================================
// KAMERA- UND SZENEN-BUENDEL
// =================================================================

// Camera buendelt die perspektivischen Parameter und die Blickrichtung.
type Camera struct {
	// Pos: Position der Kamera in WELTkoordinaten (Augpunkt).
	Pos Vec3
	// Target: Punkt in WELTkoordinaten, auf den die Kamera blickt.
	Target Vec3
	// Up: Referenz-"Oben"-Vektor der Kamera in WELTkoordinaten.
	Up Vec3
	// FovY: vertikales Blickfeld in RADIANT.
	FovY float64
	// Near: Abstand der vorderen Clipping-Ebene.
	Near float64
	// Far: Abstand der hinteren Clipping-Ebene.
	Far float64
}

// View liefert die Look-At-Matrix der Kamera.
func (c Camera) View() Matrix4x4 { return LookAtMatrix(c.Pos, c.Target, c.Up) }

// Projection liefert die perspektivische Projektionsmatrix.
func (c Camera) Projection(aspect float64) Matrix4x4 {
	return PerspectiveMatrix(c.FovY, aspect, c.Near, c.Far)
}

// Fog buendelt die Tiefen-Nebel-Parameter. Enabled == false -> kein Nebel.
type Fog struct {
	Enabled bool
	Near    float64
	Far     float64
	Color   ColorState
}

// Scene buendelt Kamera + Welt-Attribute (Licht, Nebel, Hintergrund).
type Scene struct {
	// Cam: Kamera der Szene.
	Cam Camera
	// SunDir: Richtung der Sonne in WELTkoordinaten.
	SunDir Vec3
	// SceneFog: Tiefen-Nebel.
	SceneFog Fog
	// Background: Fuellfarbe des Fensters vor dem Zeichnen (RGBA, 0..1).
	Background ColorState
}

// NewScene liefert eine Szene mit den Standardwerten (Kamera 50/100/200,
// Sonne 0.5/1/0.3, Nebel 100..400, dunkelgrauer Hintergrund).
func NewScene() Scene {
	return Scene{
		Cam: Camera{
			Pos:    Vec3{50, 100, 200},
			Target: Vec3{0, 0, 0},
			Up:     Vec3{0, 1, 0},
			FovY:   1.2,
			Near:   0.1,
			Far:    1000,
		},
		SunDir:     Vec3{0.5, 1.0, 0.3},
		SceneFog:   Fog{Enabled: true, Near: 100, Far: 400, Color: ColorState{0.25, 0.25, 0.25, 1}},
		Background: ColorState{40 / 255.0, 40 / 255.0, 40 / 255.0, 1},
	}
}
