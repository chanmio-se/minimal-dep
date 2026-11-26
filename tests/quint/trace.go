// Package quint replays Quint-generated ITF traces against the Go code.
// The traces are checked in under traces/; regenerate them with the commands
// documented in docs/quint.md.
package quint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type trace struct {
	States []state `json:"states"`
}

type state struct {
	Action    string          `json:"mbt::actionTaken"`
	Picks     map[string]pick `json:"mbt::nondetPicks"`
	Last      outcome         `json:"last"`
	Sessions  kvMap           `json:"sessions"`
	Tasks     kvMap           `json:"tasks"`
	Done      stringSet       `json:"done"`
	Cookie    string          `json:"cookie"`
	Submitted string          `json:"submitted"`
}

type outcome struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
	Task    string `json:"task"`
	Token   string `json:"token"`
}

// pick is an ITF option: {"tag": "Some"/"None", "value": ...}.
type pick struct {
	Tag   string          `json:"tag"`
	Value json.RawMessage `json:"value"`
}

type kvMap struct {
	Pairs [][2]string `json:"#map"`
}

func (m kvMap) get(key string) string {
	for _, pair := range m.Pairs {
		if pair[0] == key {
			return pair[1]
		}
	}
	return ""
}

type stringSet struct {
	Items []string `json:"#set"`
}

func (s stringSet) contains(item string) bool {
	for _, value := range s.Items {
		if value == item {
			return true
		}
	}
	return false
}

func loadTrace(t *testing.T, name string) trace {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("traces", name+".itf.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tr trace
	if err := json.Unmarshal(data, &tr); err != nil {
		t.Fatalf("decode %s trace: %v", name, err)
	}
	if len(tr.States) == 0 || tr.States[0].Action != "init" {
		t.Fatalf("%s trace does not start with init", name)
	}
	return tr
}

func pickString(t *testing.T, step int, s state, name string) string {
	t.Helper()
	value, ok := s.Picks[name]
	if !ok || value.Tag != "Some" {
		t.Fatalf("step %d: missing nondet pick %q", step, name)
	}
	var result string
	if err := json.Unmarshal(value.Value, &result); err != nil {
		t.Fatalf("step %d: pick %q is not a string: %v", step, name, err)
	}
	return result
}

func assertCode(t *testing.T, step int, want string, got string) {
	t.Helper()
	if want != got {
		t.Fatalf("step %d: result code = %q, want %q (model)", step, got, want)
	}
}
