package desktop

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTestPatternCapture(t *testing.T) {
	src := NewTestPatternSource(320, 180)
	defer src.Close()
	frame, err := src.Capture()
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if len(frame) < 100 {
		t.Fatalf("frame too small: %d bytes", len(frame))
	}
	// JPEG magic bytes.
	if frame[0] != 0xFF || frame[1] != 0xD8 {
		t.Fatal("not a JPEG")
	}
}

func TestMJPEGStream(t *testing.T) {
	src := NewTestPatternSource(160, 90)
	streamer := NewStreamer(src, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/api/v1/desktop/stream", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		streamer.ServeHTTP(rec, req)
		close(done)
	}()
	<-done

	body := rec.Body.Bytes()
	if !bytes.Contains(body, []byte("--frame")) {
		t.Fatal("no MJPEG boundary found")
	}
	if !bytes.Contains(body, []byte("Content-Type: image/jpeg")) {
		t.Fatal("no JPEG content type")
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "multipart/x-mixed-replace; boundary=frame" {
		t.Fatalf("wrong content type: %q", ct)
	}
}
