package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeGetNotebook_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetNotebook("5a088119-891d-4e93-982a-1869abe71a9e", "en")
	if rpcID != "HcT8bb" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != `["notebooks/5a088119-891d-4e93-982a-1869abe71a9e",["en"],0]` {
		t.Fatalf("payload = %s", payload)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeGetNotebook_PrefixAndDefaultLang(t *testing.T) {
	_, payload := EncodeGetNotebook("notebooks/abc", "")
	if payload != `["notebooks/abc",["en"],0]` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestDecodeGetNotebook_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "get_notebook_basic.txt", "HcT8bb")
	nb, err := DecodeGetNotebook(body)
	if err != nil {
		t.Fatal(err)
	}
	if nb == nil {
		t.Fatal("notebook is nil")
	}
	if nb.ResourceName != "notebooks/5a088119-891d-4e93-982a-1869abe71a9e" {
		t.Errorf("ResourceName = %q", nb.ResourceName)
	}
	if nb.Title != "usenix" {
		t.Errorf("Title = %q", nb.Title)
	}
	if nb.Emoji != "🚀" {
		t.Errorf("Emoji = %q", nb.Emoji)
	}
	if nb.CreatedUnix != 1769629487 {
		t.Errorf("CreatedUnix = %d", nb.CreatedUnix)
	}
	if nb.UpdatedUnix != 1789226064 {
		t.Errorf("UpdatedUnix = %d", nb.UpdatedUnix)
	}
	if len(nb.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(nb.Sources))
	}
	src := nb.Sources[0]
	if src.ResourceName != "notebooks/5a088119-891d-4e93-982a-1869abe71a9e/sources/fa1beeca-4f0c-4fa9-af38-d853d23172af" {
		t.Errorf("Source.ResourceName = %q", src.ResourceName)
	}
	if src.FileName != "atc22-li-zijun-rund.pdf" {
		t.Errorf("Source.FileName = %q", src.FileName)
	}
	if src.MimeType != "application/pdf" {
		t.Errorf("Source.MimeType = %q", src.MimeType)
	}
	if src.UploadedUnix != 1769629502 {
		t.Errorf("Source.UploadedUnix = %d", src.UploadedUnix)
	}
}

func TestDecodeGetNotebook_MultiSourceFixture(t *testing.T) {
	body := rpcFixtureBody(t, "get_notebook_multi.txt", "HcT8bb")
	nb, err := DecodeGetNotebook(body)
	if err != nil {
		t.Fatal(err)
	}
	if nb == nil {
		t.Fatal("notebook is nil")
	}
	if nb.Title != "GeFei" {
		t.Errorf("Title = %q", nb.Title)
	}
	if len(nb.Sources) != 6 {
		t.Fatalf("Sources len = %d, want 6", len(nb.Sources))
	}
	// URL sources carry the URL as the file name with a text/html mime.
	var urlSource *NotebookSource
	for i := range nb.Sources {
		if nb.Sources[i].MimeType == "text/html" {
			urlSource = &nb.Sources[i]
			break
		}
	}
	if urlSource == nil {
		t.Fatal("no text/html source found")
	}
	if urlSource.FileName != "https://indiehackertools.net/tools/gefei" {
		t.Errorf("url source FileName = %q", urlSource.FileName)
	}
}

func TestDecodeGetNotebook_Empty(t *testing.T) {
	nb, err := DecodeGetNotebook([]byte("[]"))
	if err != nil || nb != nil {
		t.Fatalf("DecodeGetNotebook([]) = %v, %v", nb, err)
	}
}
