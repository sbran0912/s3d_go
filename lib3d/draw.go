package lib3d

// ====================================================================
// ZEICHEN-FASSADE (Port von lib-render.ts)
//
// Fassade fuer das Zeichnen. Die Szene ruft ausschliesslich Draw*() auf;
// Farbe/Linienbreite/Punktgroesse kommen als Options-Objekt mit.
// ====================================================================

// PrimitiveStyle sind Zeichen-Optionen. Nicht gesetzte Felder (ColorInput.Set
// == false bzw. 0) ueberschreiben den aktuellen Low-Level-Zustand nicht.
type PrimitiveStyle struct {
	Stroke    ColorInput // Linienfarbe (unbeleuchtet)
	Fill      ColorInput // Fuellfarbe  (beleuchtet)
	LineWidth float64
	PointSize float64
}

// ShapeStyle erweitert PrimitiveStyle um den Zeichenmodus (und optionale
// Parameter fuer Rect/Circle).
type ShapeStyle struct {
	Stroke    ColorInput
	Fill      ColorInput
	LineWidth float64
	PointSize float64
	Mode      DrawStyle
	Z         float64
	Segments  int
}

// applyStyle uebernimmt die gesetzten Felder in den Low-Level-Zeichenzustand.
func applyStyle(stroke, fill ColorInput, lineWidth, pointSize float64) {
	if stroke.Set {
		StrokeColor(stroke.Color)
	}
	if fill.Set {
		FillColor(fill.Color)
	}
	if lineWidth != 0 {
		StrokeWidth(lineWidth)
	}
	if pointSize != 0 {
		PointSize(pointSize)
	}
}

// ====================================================================
// MESHES
// ====================================================================

// DrawMesh zeichnet ein Mesh in lokalen Koordinaten und wendet dabei die
// uebergebene Modell-Matrix an (Default: Identity). Fill beleuchtet, Lines
// als Drahtgitter. Die Geometrie entsteht ausserhalb ueber Make*Mesh und
// kann so auch fuer Kollision/Raycasting wiederverwendet werden.
func DrawMesh(mesh Mesh, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	m := modelOr(model)
	if style.Mode == StyleFill || style.Mode == StyleBoth {
		submitTriangles(mesh.Triangles, m)
	}
	if style.Mode == StyleStroke || style.Mode == StyleBoth {
		submitLines(mesh.Lines, m)
	}
}

// ====================================================================
// GPU-INSTANCING
// ====================================================================

// DrawMeshInstanced zeichnet dieselbe Geometrie mehrfach in einem Draw-Call.
// Jede Instance bringt ihre eigene Modell-Matrix und Farbe mit; style
// bestimmt Fill/Stroke und die Linienbreite. (Farbe aus style wird ignoriert.)
func DrawMeshInstanced(mesh Mesh, style ShapeStyle, instances []Instance) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	submitInstanced(mesh.Triangles, mesh.Lines, style.Mode, instances)
}

// ====================================================================
// IMMEDIATE-PRIMITIVES
// ====================================================================

// DrawPoint zeichnet einen 3D-Punkt bei (x,y,z).
func DrawPoint(x, y, z float64, style PrimitiveStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Point(x, y, z, modelOr(model))
}

// DrawLine zeichnet eine 3D-Linie von (x1,y1,z1) nach (x2,y2,z2).
func DrawLine(x1, y1, z1, x2, y2, z2 float64, style PrimitiveStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Line(x1, y1, z1, x2, y2, z2, modelOr(model))
}

// DrawTriangle zeichnet ein 3D-Dreieck (Mode: 0=stroke | 1=fill | 2=beide).
func DrawTriangle(x1, y1, z1, x2, y2, z2, x3, y3, z3 float64, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Triangle(x1, y1, z1, x2, y2, z2, x3, y3, z3, style.Mode, modelOr(model))
}

// DrawShape zeichnet ein 3D-Viereck mit 4 beliebigen Punkten.
func DrawShape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4 float64, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Shape(x1, y1, z1, x2, y2, z2, x3, y3, z3, x4, y4, z4, style.Mode, modelOr(model))
}

// DrawRect zeichnet ein achsenparalleles Rechteck in der XY-Ebene bei style.Z.
func DrawRect(x, y, w, h float64, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Rect(x, y, w, h, style.Mode, style.Z, modelOr(model))
}

// DrawCircle zeichnet einen 3D-Kreis (Ring) in der XY-Ebene bei z
// (style.Segments; 0 -> 64).
func DrawCircle(x, y, z, radius float64, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	segments := style.Segments
	if segments == 0 {
		segments = 64
	}
	Circle(x, y, z, radius, style.Mode, segments, modelOr(model))
}

// DrawPolygon zeichnet ein beliebiges 3D-Polygon als flaches Array
// [x0,y0,z0, x1,y1,z1, ...] (Mode: 0=stroke | 1=fill | 2=beide).
func DrawPolygon(pts []float64, style ShapeStyle, model ...Matrix4x4) {
	applyStyle(style.Stroke, style.Fill, style.LineWidth, style.PointSize)
	Polygon(pts, style.Mode, modelOr(model))
}
