package ui

// Shell Phase (Playable Game Shell): Camera transform for the playing state.
// X and Y are the world tile coordinates rendered at the center of the screen;
// Zoom scales the 16px base tile. The math mirrors the entity pass in
// state_playing.go Draw(): sx = sw/2 + (wx - camX) * tileSize.

const (
	baseTileSize = 16.0
	minZoom      = 0.25
	maxZoom      = 4.0
)

// Camera holds the strategic/action view transform state.
type Camera struct {
	X    float64 // World tile X at screen center
	Y    float64 // World tile Y at screen center
	Zoom float64 // Multiplier over the 16px base tile (clamped 0.25..4.0)
}

// TileSize returns the on-screen pixel size of one world tile.
func (c *Camera) TileSize() float64 {
	return baseTileSize * c.Zoom
}

// WorldToScreen projects world tile coordinates to screen pixels for a
// viewport of sw x sh pixels.
func (c *Camera) WorldToScreen(wx, wy float32, sw, sh int) (float64, float64) {
	ts := c.TileSize()
	sx := float64(sw)/2 + (float64(wx)-c.X)*ts
	sy := float64(sh)/2 + (float64(wy)-c.Y)*ts
	return sx, sy
}

// ScreenToWorld inverts WorldToScreen, mapping a screen pixel back to world
// tile coordinates for a viewport of sw x sh pixels.
func (c *Camera) ScreenToWorld(sx, sy, sw, sh int) (float32, float32) {
	ts := c.TileSize()
	wx := c.X + (float64(sx)-float64(sw)/2)/ts
	wy := c.Y + (float64(sy)-float64(sh)/2)/ts
	return float32(wx), float32(wy)
}

// ZoomBy multiplies the current zoom by factor and clamps the result.
func (c *Camera) ZoomBy(factor float64) {
	c.Zoom *= factor
	c.ClampZoom()
}

// ClampZoom bounds Zoom to the playable 0.25..4.0 range.
func (c *Camera) ClampZoom() {
	if c.Zoom < minZoom {
		c.Zoom = minZoom
	}
	if c.Zoom > maxZoom {
		c.Zoom = maxZoom
	}
}
