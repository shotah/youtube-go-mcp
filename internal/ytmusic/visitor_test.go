package ytmusic

import "testing"

func TestVisitorIDFromHTML(t *testing.T) {
	html := `<html><script>ytcfg.set({"VISITOR_DATA":"Cgtabc123","INNERTUBE_CONTEXT":{}});</script></html>`
	if got := visitorIDFromHTML(html); got != "Cgtabc123" {
		t.Fatalf("got %q", got)
	}
	loose := `var x = {"VISITOR_DATA":"Cgloose"};`
	if got := visitorIDFromHTML(loose); got != "Cgloose" {
		t.Fatalf("loose got %q", got)
	}
	if visitorIDFromHTML(`<html></html>`) != "" {
		t.Fatal("expected empty")
	}
}
