package natspec

import (
	"crypto/rand"
	"io/ioutil"
	"testing"
)

func readJSON() (docJSON, defJSON []byte, err error) {
	if docJSON, err = ioutil.ReadFile("doc.json"); err != nil {
		return
	}
	defJSON, err = ioutil.ReadFile("def.json")
	return
}

func TestNotice(t *testing.T) {
	var docJSON, defJSON []byte
	var err error
	if docJSON, defJSON, err = readJSON(); err != nil {
		t.Errorf("unable to read contract definition json files: %v", err)
		return
	}
	natspec, err := New(docJSON, defJSON, nil)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	var address = make([]byte, 160)
	rand.Read(address)
	notice, err := natspec.Notice("send", address, 1000)
	if err != nil {
		return
	}
	expected := "hello"
	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	}
}
