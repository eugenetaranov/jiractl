package jira

import (
	"encoding/json"
	"strings"
)

// adfToText renders an Atlassian Document Format node as plain text:
// paragraphs and headings become lines, list items get "- ", hard breaks
// become newlines. Unknown nodes contribute their children's text.
func adfToText(node map[string]interface{}) string {
	var sb strings.Builder
	var walk func(n map[string]interface{}, prefix string)
	walk = func(n map[string]interface{}, prefix string) {
		typ, _ := n["type"].(string)
		switch typ {
		case "text":
			s, _ := n["text"].(string)
			sb.WriteString(s)
			return
		case "hardBreak":
			sb.WriteString("\n")
			return
		case "mention", "emoji":
			if attrs, ok := n["attrs"].(map[string]interface{}); ok {
				s, _ := attrs["text"].(string)
				sb.WriteString(s)
			}
			return
		}

		children, _ := n["content"].([]interface{})
		childPrefix := prefix
		if typ == "listItem" {
			sb.WriteString(prefix + "- ")
			childPrefix = prefix + "  "
		}
		for _, c := range children {
			if cm, ok := c.(map[string]interface{}); ok {
				walk(cm, childPrefix)
			}
		}
		switch typ {
		case "paragraph", "heading", "codeBlock", "blockquote", "rule":
			sb.WriteString("\n")
			if prefix == "" {
				sb.WriteString("\n")
			}
		}
	}
	walk(node, "")
	return strings.TrimRight(sb.String(), "\n ")
}

// flattenADFDescriptions rewrites each issue's fields.description from an
// ADF object (Jira Cloud v3) to plain text, so go-jira can decode it.
func flattenADFDescriptions(raw []json.RawMessage) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, len(raw))
	for i, r := range raw {
		var issue map[string]interface{}
		if err := json.Unmarshal(r, &issue); err != nil {
			return nil, err
		}
		if fields, ok := issue["fields"].(map[string]interface{}); ok {
			if doc, ok := fields["description"].(map[string]interface{}); ok {
				fields["description"] = adfToText(doc)
			}
		}
		b, err := json.Marshal(issue)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}
