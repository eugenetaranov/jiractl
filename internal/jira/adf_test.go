package jira

import (
	"encoding/json"
	"testing"
)

func TestADFToText(t *testing.T) {
	var doc map[string]interface{}
	err := json.Unmarshal([]byte(`{"type":"doc","content":[
		{"type":"paragraph","content":[{"type":"text","text":"Hello "},{"type":"text","text":"world"}]},
		{"type":"bulletList","content":[
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},
			{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]},
		{"type":"paragraph","content":[{"type":"text","text":"a"},{"type":"hardBreak"},{"type":"text","text":"b"}]}]}`), &doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "Hello world\n\n- one\n- two\na\nb"
	if got := adfToText(doc); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
