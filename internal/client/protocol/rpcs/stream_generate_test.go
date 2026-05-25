package rpcs

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func loadProtocolTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return bytes.TrimSpace(data)
}

func marshalCompact(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func fixtureOpts() EncodeStreamGenerateOpts {
	return EncodeStreamGenerateOpts{
		Prompt:        "Sample user prompt",
		Language:      "en",
		UUID:          "00000000-0000-0000-0000-000000000001",
		EntropyToken:  "!" + string(bytes.Repeat([]byte("A"), 4096)),
		HexUUID:       "00000000000000000000000000000001",
		ModelSelector: 1,
	}
}

func TestEncodeStreamGenerate_WireParity_Basic(t *testing.T) {
	got := marshalCompact(t, EncodeStreamGenerate(fixtureOpts()))
	want := loadProtocolTestdata(t, "stream_generate_basic_inner_req.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("wire mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestEncodeStreamGenerate_WireParity_Upload(t *testing.T) {
	opts := fixtureOpts()
	opts.Uploads = []FileRef{{UploadID: "upload_000000000000001", MimeType: "image/png", FileName: "sample.png"}}
	got := marshalCompact(t, EncodeStreamGenerate(opts))
	want := loadProtocolTestdata(t, "stream_generate_with_upload_inner_req.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("wire mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestEncodeStreamGenerate_WireParity_Video(t *testing.T) {
	opts := fixtureOpts()
	opts.Mode = "video"
	got := marshalCompact(t, EncodeStreamGenerate(opts))
	want := loadProtocolTestdata(t, "stream_generate_video_inner_req.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("wire mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestEncodeStreamGenerate_WireParity_Music(t *testing.T) {
	opts := fixtureOpts()
	opts.Mode = "music"
	got := marshalCompact(t, EncodeStreamGenerate(opts))
	want := loadProtocolTestdata(t, "stream_generate_music_inner_req.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("wire mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestEncodeStreamGenerate_WireParity_DeepResearch(t *testing.T) {
	opts := fixtureOpts()
	opts.DeepResearch = true
	got := marshalCompact(t, EncodeStreamGenerate(opts))
	want := loadProtocolTestdata(t, "stream_generate_deep_research_inner_req.json")
	if !bytes.Equal(got, want) {
		t.Fatalf("wire mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestDecodeStreamGenerateFrame_Basic(t *testing.T) {
	content := make([]any, 26)
	content[1] = []any{"c_abc", "r_def"}
	content[4] = []any{[]any{"rc_ghi", []any{"hello"}}}
	content[25] = "ctx"
	contentJSON, _ := json.Marshal(content)
	envelope := []any{[]any{"wrb.fr", nil, string(contentJSON)}}

	out, err := DecodeStreamGenerateFrame(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.Text != "hello" || !out.Done || out.RCid != "rc_ghi" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestDecodeStreamGenerateFrame_EnvelopeError(t *testing.T) {
	_, err := DecodeStreamGenerateFrame([]any{"wrb.fr", nil, "", nil, nil, []any{float64(1052)}})
	if e, ok := err.(*EnvelopeError); !ok || e.Code != 1052 {
		t.Fatalf("err = %#v, want EnvelopeError 1052", err)
	}
}
