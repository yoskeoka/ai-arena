package artifactbundle

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestReadAcceptsFixedGameBundle(t *testing.T) {
	data := testZIP(t, `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`)
	bundle, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if bundle.Digest == "" || bundle.Manifest.GameID != "test" {
		t.Fatalf("bundle = %+v", bundle)
	}
}

func TestReadRejectsUndeclaredEntry(t *testing.T) {
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for name, value := range map[string]string{"manifest.json": `{"schema_version":"arena-bundle/v1","artifact_kind":"game","runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`, "module.wasm": "\x00asm", "extra.txt": "no"} {
		entry, _ := writer.Create(name)
		_, _ = entry.Write([]byte(value))
	}
	_ = writer.Close()
	if _, err := Read(out.Bytes()); err == nil {
		t.Fatal("Read accepted undeclared entry")
	}
}

func testZIP(t *testing.T, manifest string) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	entry, err := writer.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte(manifest))
	entry, err = writer.Create("module.wasm")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte{0, 97, 115, 109, 1, 0, 0, 0})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
