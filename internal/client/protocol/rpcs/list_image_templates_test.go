package rpcs

import "testing"

func TestEncodeListImageTemplates_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListImageTemplates()
	if rpcID != "XhaU0b" {
		t.Fatalf("rpcID = %q, want XhaU0b", rpcID)
	}
	if payload != "[4,[2],3]" {
		t.Fatalf("payload = %q", payload)
	}
}

func TestDecodeListImageTemplates_FromSampleFixture(t *testing.T) {
	templates, err := DecodeListImageTemplates(rpcFixtureBody(t, "list_image_templates_basic.txt", "XhaU0b"))
	if err != nil {
		t.Fatalf("DecodeListImageTemplates: %v", err)
	}
	if len(templates) != 3 {
		t.Fatalf("templates = %d, want 3", len(templates))
	}
	if templates[0].ID != "template_sample_1" {
		t.Fatalf("ID = %q", templates[0].ID)
	}
	if templates[0].Name != "Monochrome" {
		t.Fatalf("Name = %q", templates[0].Name)
	}
	if templates[0].PreviewURL != "https://lh3.googleusercontent.com/sample-template-1" {
		t.Fatalf("PreviewURL = %q", templates[0].PreviewURL)
	}
}

func TestDecodeListImageTemplates_EmptyBody(t *testing.T) {
	templates, err := DecodeListImageTemplates(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != 0 {
		t.Fatalf("templates = %d, want 0", len(templates))
	}
}

func TestDecodeListImageTemplates_MalformedJSON(t *testing.T) {
	if _, err := DecodeListImageTemplates([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
