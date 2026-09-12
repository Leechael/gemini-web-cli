package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeCreateNotebook_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeCreateNotebook("GeFei")
	if rpcID != "oMH3Zd" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	want := `[["GeFei","",null,null,null,null,null,null,0,null,1,null,null,null,null,null,[2,null,null,null,1]]]`
	if payload != want {
		t.Fatalf("payload = %s\nwant %s", payload, want)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeCreateNotebook_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "create_notebook_basic.txt", "oMH3Zd")
	resource, err := DecodeCreateNotebook(body)
	if err != nil {
		t.Fatal(err)
	}
	if resource != "notebooks/73830cf9-85d5-4fb8-993b-7469753a7c28" {
		t.Fatalf("resource = %q", resource)
	}
}

func TestEncodeAddNotebookSource_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeAddNotebookSource(
		"73830cf9-85d5-4fb8-993b-7469753a7c28",
		"00-principles.md",
		"text/markdown",
		"/contrib_service/ttl_1d/etl3n63xqda5jdxzktc5sgqhv5s3tr1789226544_AR5BTA59baXXn3m6qAZ_U9z3ICvwxVUiULHkFw2L_0luczjsgIa3Wf2G2qYN",
	)
	if rpcID != "ko3zcd" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	want := `["notebooks/73830cf9-85d5-4fb8-993b-7469753a7c28",[null,"00-principles.md",null,null,["/contrib_service/ttl_1d/etl3n63xqda5jdxzktc5sgqhv5s3tr1789226544_AR5BTA59baXXn3m6qAZ_U9z3ICvwxVUiULHkFw2L_0luczjsgIa3Wf2G2qYN","text/markdown"],null,null,null,"text/markdown"],[1,3]]`
	if payload != want {
		t.Fatalf("payload = %s\nwant %s", payload, want)
	}
}

func TestEncodeAddNotebookURLSource_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeAddNotebookURLSource(
		"73830cf9-85d5-4fb8-993b-7469753a7c28",
		"https://indiehackertools.net/tools/gefei",
	)
	if rpcID != "ko3zcd" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	want := `["notebooks/73830cf9-85d5-4fb8-993b-7469753a7c28",[null,"https://indiehackertools.net/tools/gefei",null,["https://indiehackertools.net/tools/gefei"],null,null,null,null,"text/html"],[1,3]]`
	if payload != want {
		t.Fatalf("payload = %s\nwant %s", payload, want)
	}
}

func TestDecodeAddNotebookURLSource_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "add_notebook_url_source_basic.txt", "ko3zcd")
	nb, err := DecodeAddNotebookSource(body)
	if err != nil {
		t.Fatal(err)
	}
	if nb == nil || nb.Title != "GeFei" {
		t.Fatalf("notebook = %+v", nb)
	}
}

func TestEncodeRemoveNotebookSource_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeRemoveNotebookSource("notebooks/nb-1/sources/src-1")
	if rpcID != "AptDmf" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	want := `["notebooks/nb-1/sources/src-1",[1,3]]`
	if payload != want {
		t.Fatalf("payload = %s\nwant %s", payload, want)
	}
}

func TestDecodeRemoveNotebookSource_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "remove_notebook_source_basic.txt", "AptDmf")
	if err := DecodeRemoveNotebookSource(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeAddNotebookSource_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "add_notebook_source_basic.txt", "ko3zcd")
	nb, err := DecodeAddNotebookSource(body)
	if err != nil {
		t.Fatal(err)
	}
	if nb == nil {
		t.Fatal("notebook is nil")
	}
	if nb.ResourceName != "notebooks/73830cf9-85d5-4fb8-993b-7469753a7c28" {
		t.Errorf("ResourceName = %q", nb.ResourceName)
	}
	if nb.Title != "GeFei" {
		t.Errorf("Title = %q", nb.Title)
	}
	if len(nb.Sources) != 1 {
		t.Fatalf("Sources len = %d, want 1", len(nb.Sources))
	}
	src := nb.Sources[0]
	if src.ResourceName != "notebooks/73830cf9-85d5-4fb8-993b-7469753a7c28/sources/2c5b9e09-265a-4fd0-a5dc-738b30f7b19d" {
		t.Errorf("Source.ResourceName = %q", src.ResourceName)
	}
	if src.FileName != "00-principles.md" {
		t.Errorf("Source.FileName = %q", src.FileName)
	}
	if src.UploadedUnix != 1789226547 {
		t.Errorf("Source.UploadedUnix = %d", src.UploadedUnix)
	}
}
