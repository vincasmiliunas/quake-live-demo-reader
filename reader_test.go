package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestDuelDemo(tst *testing.T) {
	raw, err := ioutil.ReadFile("duel.dm_73")
	if err != nil {
		tst.Errorf("Failed to load duel.dm_73")
	}
	demoState := NewDemoState()
	reader := NewDemoReader(bytes.NewReader(raw), demoState)

	gg := false
	for entry := range reader.Iterate() {
		switch entry.(type) {
		case *Command:
			gg = gg || strings.HasSuffix(entry.(*Command).Str, ` ^2gg"`)
		case *Gamestate:
		case *Snapshot:
		default:
		}
	}
	if !gg {
		tst.Errorf("Expected a gg message but did not find it.")
	}
}
