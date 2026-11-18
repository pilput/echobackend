package openrouter

import (
	"reflect"
	"testing"
)

func TestParseStreamData(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		deltas []Delta
		usage  *Usage
		err    string
	}{
		{
			name:   "content chunk",
			data:   `{"choices":[{"index":0,"delta":{"role":"assistant","content":"Hi"}}]}`,
			deltas: []Delta{{Content: "Hi"}},
		},
		{
			name: "null content and reasoning are skipped",
			data: `{"choices":[{"delta":{"content":null,"reasoning":null}}]}`,
		},
		{
			name: "reasoning_details text wins over duplicate plain reasoning",
			data: `{"choices":[{"delta":{"content":"","reasoning":"think","reasoning_details":[
				{"type":"reasoning.text","text":"think","format":"anthropic-claude-v1","index":0},
				{"type":"reasoning.encrypted","data":"xyz"}]}}]}`,
			deltas: []Delta{{Reasoning: "think"}},
		},
		{
			name:   "reasoning summary",
			data:   `{"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.summary","summary":"sum"}]}}]}`,
			deltas: []Delta{{Reasoning: "sum"}},
		},
		{
			name:   "plain reasoning fallback",
			data:   `{"choices":[{"delta":{"content":"","reasoning":"step 1"}}]}`,
			deltas: []Delta{{Reasoning: "step 1"}},
		},
		{
			name: "encrypted-only reasoning emits nothing",
			data: `{"choices":[{"delta":{"reasoning_details":[{"type":"reasoning.encrypted","data":"x"}]}}]}`,
		},
		{
			name: "usage chunk",
			data: `{"choices":[{"index":0,"delta":{"content":"","role":"assistant"},"finish_reason":"stop"}],
				"usage":{"prompt_tokens":3,"completion_tokens":5,"total_tokens":8}}`,
			usage: &Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
		},
		{
			name: "mid-stream error",
			data: `{"error":{"code":502,"message":"Provider disconnected unexpectedly"},
				"choices":[{"index":0,"delta":{"content":""},"finish_reason":"error"}]}`,
			err: "Provider disconnected unexpectedly",
		},
		{
			name: "invalid json is ignored",
			data: "not json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseStreamData(tc.data)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !reflect.DeepEqual(got.deltas, tc.deltas) {
				t.Errorf("deltas = %#v, want %#v", got.deltas, tc.deltas)
			}
			if !reflect.DeepEqual(got.usage, tc.usage) {
				t.Errorf("usage = %#v, want %#v", got.usage, tc.usage)
			}
		})
	}
}
