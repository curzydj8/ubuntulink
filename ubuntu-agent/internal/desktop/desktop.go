// Package desktop provides MJPEG desktop streaming.
// Phase 4: pluggable CaptureSource; TestPatternSource validates the pipeline.
// PortalSource (real Wayland capture) is documented in docs/DESKTOP_STREAMING.md.
package desktop

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"time"
)

// CaptureSource produces JPEG frames.
type CaptureSource interface {
	// Capture returns a JPEG-encoded frame.
	Capture() ([]byte, error)
	// Close releases resources.
	Close() error
}

// TestPatternSource generates a timestamped test image.
// It validates the MJPEG pipeline without requiring a graphical session.
type TestPatternSource struct {
	width, height int
}

func NewTestPatternSource(width, height int) *TestPatternSource {
	return &TestPatternSource{width: width, height: height}
}

func (t *TestPatternSource) Capture() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, t.width, t.height))
	// Dark background.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{20, 20, 30, 255}}, image.Point{}, draw.Src)
	// Color bars.
	barW := t.width / 6
	colors := []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255},
		{255, 255, 0, 255}, {0, 255, 255, 255}, {255, 0, 255, 255},
	}
	for i, c := range colors {
		x0 := i * barW
		draw.Draw(img, image.Rect(x0, 0, x0+barW, t.height/2),
			&image.Uniform{c}, image.Point{}, draw.Src)
	}
	// Timestamp text (simple pixel blocks, not a font).
	ts := time.Now().Format("15:04:05")
	drawTimestamp(img, ts, 20, t.height-40)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TestPatternSource) Close() error { return nil }

// drawTimestamp draws digits using 3x5 pixel blocks.
func drawTimestamp(img *image.RGBA, s string, x0, y0 int) {
	digits := map[rune][5]string{
		'0': {"###", "# #", "# #", "# #", "###"},
		'1': {" # ", "## ", " # ", " # ", "###"},
		'2': {"###", "  #", "###", "#  ", "###"},
		'3': {"###", "  #", "###", "  #", "###"},
		'4': {"# #", "# #", "###", "  #", "  #"},
		'5': {"###", "#  ", "###", "  #", "###"},
		'6': {"###", "#  ", "###", "# #", "###"},
		'7': {"###", "  #", "  #", "  #", "  #"},
		'8': {"###", "# #", "###", "# #", "###"},
		'9': {"###", "# #", "###", "  #", "###"},
		':': {"   ", " # ", "   ", " # ", "   "},
	}
	white := color.RGBA{255, 255, 255, 255}
	x := x0
	for _, ch := range s {
		glyph, ok := digits[ch]
		if !ok {
			x += 8
			continue
		}
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if glyph[row][col] == '#' {
					for dy := 0; dy < 3; dy++ {
						for dx := 0; dx < 3; dx++ {
							img.Set(x+col*3+dx, y0+row*3+dy, white)
						}
					}
				}
			}
		}
		x += 14
	}
}

// Streamer serves MJPEG over HTTP.
type Streamer struct {
	source CaptureSource
	fps    int
}

func NewStreamer(source CaptureSource, fps int) *Streamer {
	if fps <= 0 {
		fps = 5
	}
	return &Streamer{source: source, fps: fps}
}

// ServeHTTP implements multipart/x-mixed-replace MJPEG streaming.
func (s *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	ticker := time.NewTicker(time.Second / time.Duration(s.fps))
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			frame, err := s.source.Capture()
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
			w.Write(frame)
			fmt.Fprintf(w, "\r\n")
			flusher.Flush()
		}
	}
}
