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

func TestReadRejectsUnsafeAndInvalidWASMEntries(t *testing.T) {
	for _, tc := range []struct {
		name, module   string
		manifestModule string
	}{
		{name: "path", module: "module.wasm", manifestModule: "../module.wasm"},
		{name: "version", module: "module.wasm", manifestModule: "module.wasm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			writer := zip.NewWriter(&out)
			entry, _ := writer.Create("manifest.json")
			_, _ = entry.Write([]byte(`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"` + tc.manifestModule + `"}}`))
			entry, _ = writer.Create(tc.module)
			wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0}
			if tc.name == "version" {
				wasm[4] = 2
			}
			_, _ = entry.Write(wasm)
			_ = writer.Close()
			if _, err := Read(out.Bytes()); err == nil {
				t.Fatal("Read accepted invalid bundle")
			}
		})
	}
}

func TestReadRejectsExplicitZeroResourceLimits(t *testing.T) {
	for _, manifest := range []string{
		`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":0}}`,
		`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm","timeout_ms":0}}`,
		`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","rulesets":[{"ruleset_version":"1","player_count":0}],"runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`,
		`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","rulesets":[{"ruleset_version":"1","max_active_bots_per_owner":0}],"runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`,
	} {
		if _, err := Read(testZIP(t, manifest)); err == nil {
			t.Fatalf("Read accepted an explicit zero resource limit: %s", manifest)
		}
	}
}

func TestReadRejectsAIWithoutAIID(t *testing.T) {
	data := testZIP(t, `{"schema_version":"arena-bundle/v1","artifact_kind":"ai","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`)
	if _, err := Read(data); err == nil {
		t.Fatal("Read accepted AI bundle without ai_id")
	}
}

func TestValidateWASMImportsRejectsNonWASIImportForEveryKind(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind byte
		desc []byte
	}{
		{name: "function", kind: 0, desc: []byte{0}},
		{name: "table", kind: 1, desc: []byte{0x70, 0, 0}},
		{name: "memory", kind: 2, desc: []byte{0, 0}},
		{name: "global", kind: 3, desc: []byte{0x7f, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imports := append([]byte{1, 3, 'e', 'n', 'v', 1, 'x', tc.kind}, tc.desc...)
			module := append([]byte{0, 97, 115, 109, 1, 0, 0, 0}, 2, byte(len(imports)))
			module = append(module, imports...)
			if err := validateWASMImports(module); err == nil {
				t.Fatalf("validateWASMImports accepted a non-WASI %s import", tc.name)
			}
		})
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
