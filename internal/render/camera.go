package render

// Phase 08.1: Window Management & Camera
// Camera math calculates arbitrary matrices translating the MapGrid bounds into specific pixel offsets.

// Camera handles panning and zooming across the 2D grid.
type Camera struct {
	X    float64
	Y    float64
	Zoom float64
}

// NewCamera creates a new default camera instance.
func NewCamera() *Camera {
	return &Camera{
		X:    0,
		Y:    0,
		Zoom: 1.0,
	}
}

// WorldToScreen translates abstract world coordinates to rendering pixel coordinates.
// Applies both translation (panning) and scaling (zooming).
func (c *Camera) WorldToScreen(worldX, worldY float64) (screenX, screenY float64) {
	screenX = (worldX - c.X) * c.Zoom
	screenY = (worldY - c.Y) * c.Zoom
	return screenX, screenY
}

// ScreenToWorld translates screen pixel coordinates back into abstract world coordinates.
// This is critical for mouse clicking mapping correctly onto the grid.
func (c *Camera) ScreenToWorld(screenX, screenY float64) (worldX, worldY float64) {
	worldX = (screenX / c.Zoom) + c.X
	worldY = (screenY / c.Zoom) + c.Y
	return worldX, worldY
}
