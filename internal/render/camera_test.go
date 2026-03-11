package render

import (
	"math"
	"testing"
)

// Phase 08.1: Window Management & Camera Tests
// Tests map bounds to pixel boundaries via arbitrary matrix transformations

const epsilon = 1e-9

func TestWorldToScreen(t *testing.T) {
	cam := NewCamera()

	// Default camera state
	sx, sy := cam.WorldToScreen(100.0, 50.0)
	if sx != 100.0 || sy != 50.0 {
		t.Fatalf("Expected screen (100.0, 50.0), got (%f, %f)", sx, sy)
	}

	// Panned camera
	cam.X = 20.0
	cam.Y = 10.0
	sx, sy = cam.WorldToScreen(100.0, 50.0)
	if sx != 80.0 || sy != 40.0 {
		t.Fatalf("Expected screen (80.0, 40.0), got (%f, %f)", sx, sy)
	}

	// Zoomed camera
	cam.Zoom = 2.0
	sx, sy = cam.WorldToScreen(100.0, 50.0)
	if sx != 160.0 || sy != 80.0 {
		t.Fatalf("Expected screen (160.0, 80.0), got (%f, %f)", sx, sy)
	}
}

func TestScreenToWorld(t *testing.T) {
	cam := NewCamera()

	// Default camera state
	wx, wy := cam.ScreenToWorld(100.0, 50.0)
	if wx != 100.0 || wy != 50.0 {
		t.Fatalf("Expected world (100.0, 50.0), got (%f, %f)", wx, wy)
	}

	// Panned camera
	cam.X = 20.0
	cam.Y = 10.0
	wx, wy = cam.ScreenToWorld(80.0, 40.0)
	if wx != 100.0 || wy != 50.0 {
		t.Fatalf("Expected world (100.0, 50.0), got (%f, %f)", wx, wy)
	}

	// Zoomed camera
	cam.Zoom = 2.0
	wx, wy = cam.ScreenToWorld(160.0, 80.0)
	if wx != 100.0 || wy != 50.0 {
		t.Fatalf("Expected world (100.0, 50.0), got (%f, %f)", wx, wy)
	}
}

func TestCameraPanAndZoom(t *testing.T) {
	cam := NewCamera()
	cam.X = -50.5
	cam.Y = 20.25
	cam.Zoom = 0.5

	wx, wy := 150.0, 75.0

	// Project to screen
	sx, sy := cam.WorldToScreen(wx, wy)

	// Back to world
	nwx, nwy := cam.ScreenToWorld(sx, sy)

	if math.Abs(nwx-wx) > epsilon || math.Abs(nwy-wy) > epsilon {
		t.Fatalf("Bidirectional projection failed. Expected (%f, %f), got (%f, %f)", wx, wy, nwx, nwy)
	}
}
