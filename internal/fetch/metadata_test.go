package fetch

import (
	"net/url"
	"testing"
)

func TestImagesAreTheirOwnPreview(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	for name, res := range map[string]*Response{
		"declared": {ContentType: "image/jpeg", Body: []byte("whatever")},
		"sniffed":  {ContentType: "application/octet-stream", Body: png},
		"no type":  {Body: png},
	} {
		if !isImage(res) {
			t.Errorf("%s: not recognized as an image", name)
		}
	}
	if isImage(&Response{ContentType: "text/html; charset=utf-8", Body: png}) {
		t.Error("an HTML page is not an image, whatever its bytes")
	}
	location, _ := url.Parse("https://example.org/photos/Sunset%20Beach.jpg?w=800")
	meta := imageMetadata(location)
	if meta.Title != "Sunset Beach.jpg" || meta.OgImage != location.String() || !meta.Found() {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}
