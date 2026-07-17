package main

import "testing"

func TestHeadersFromPrompts(t *testing.T) {
	h, err := headersFromPrompts(
		"SID=abc; __Secure-3PAPISID=sapisid-value; other=1",
		"0",
	)
	if err != nil {
		t.Fatal(err)
	}
	if h["cookie"] == "" || h["x-goog-authuser"] != "0" {
		t.Fatalf("got %#v", h)
	}
	if h["content-type"] != "application/json" || h["x-origin"] != "https://music.youtube.com" {
		t.Fatalf("defaults missing: %#v", h)
	}
}

func TestHeadersFromPromptsStripsPrefixes(t *testing.T) {
	h, err := headersFromPrompts(
		"cookie: SID=abc; __Secure-3PAPISID=sapisid-value",
		"x-goog-authuser: 1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if h["cookie"] != "SID=abc; __Secure-3PAPISID=sapisid-value" {
		t.Fatalf("cookie=%q", h["cookie"])
	}
	if h["x-goog-authuser"] != "1" {
		t.Fatalf("authuser=%q", h["x-goog-authuser"])
	}
}

func TestHeadersFromPromptsRequiresBoth(t *testing.T) {
	if _, err := headersFromPrompts("", "0"); err == nil {
		t.Fatal("expected error")
	}
}
