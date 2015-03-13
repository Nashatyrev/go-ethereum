package main

import (
	"fmt"
	"github.com/obscuren/otto"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
)

func TestJEthRE(t *testing.T) {
	os.RemoveAll("/tmp/eth/")
	err := os.MkdirAll("/tmp/eth/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	err = os.MkdirAll("/tmp/eth/data", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	// FIXME: this does not work ATM
	ks := crypto.NewKeyStorePlain("/tmp/eth/keys")
	ethutil.WriteFile("/tmp/eth/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/e273f01c99144c438695e10f24926dc1f9fbf62d",
		[]byte(`{"Id":"RhRXD+fNRKS4jx+7ZfEsNA==","Address":"4nPwHJkUTEOGleEPJJJtwfn79i0=","PrivateKey":"h4ACVpe74uIvi5Cg/2tX/Yrm2xdr3J7QoMbMtNX2CNc="}`))

	ethereum, err := eth.New(&eth.Config{
		DataDir:        "/tmp/eth",
		AccountManager: accounts.NewManager(ks),
	})

	if err != nil {
		t.Errorf("%v", err)
		return
	}

	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	repl := newJSRE(ethereum, assetPath)

	val, err := repl.re.Run("eth.coinbase")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	pp, err := repl.re.PrettyPrint(val)
	if err != nil {
		t.Errorf("%v", err)
	}

	if !val.IsString() {
		t.Errorf("incorrect type, expected string, got %v: %v", val, pp)
	}
	strVal, _ := val.ToString()
	expected := "0xe273f01c99144c438695e10f24926dc1f9fbf62d"
	if strVal != expected {
		t.Errorf("incorrect result, expected %s, got %v", expected, strVal)
	}

	val, err = repl.re.Run(`admin.newAccount("password")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addr, err := val.ToString()
	if err != nil {
		t.Errorf("expected string, got %v", err)
	}

	val, err = repl.re.Run("eth.accounts")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addrs, err := val.ToString()
	if err != nil {
		t.Errorf("expected string, got %v", err)
	}
	if addr != addrs {
	}
	// if lenaddr !=  {
	// 	t.Errorf("expected false (not mining), got true")
	// }

	// should get current block
	val0, err := repl.re.Run("admin.dumpBlock()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	fn := "/tmp/eth/data/blockchain.0"
	val, err = repl.re.Run("admin.export(\"" + fn + "\")")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if _, err = os.Stat(fn); err != nil {
		t.Errorf("expected no error on file, got %v", err)
	}

	ethereum, err = eth.New(&eth.Config{
		DataDir:        "/tmp/eth1",
		AccountManager: accounts.NewManager(ks),
	})
	val, err = repl.re.Run("admin.import(\"" + fn + "\")")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var val1 otto.Value

	// should get current block
	val1, err = repl.re.Run("admin.dumpBlock()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// FIXME: neither != , nor reflect.DeepEqual works, doing string comparison
	v0 := fmt.Sprintf("%v", val0)
	v1 := fmt.Sprintf("%v", val1)
	if v0 != v1 {
		t.Errorf("expected same head after export-import, got %v (!=%v)", v1, v0)
	}

	ethereum.Start()
	defer ethereum.Stop()

	val, err = repl.re.Run("eth.mining")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	var mining bool
	mining, err = val.ToBoolean()
	if err != nil {
		t.Errorf("expected boolean, got %v", err)
	}
	if mining {
		t.Errorf("expected false (not mining), got true")
	}

	val, err = repl.re.Run("admin.startMining(4)")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if !mining {
		t.Errorf("expected true (mining), got false")
	}
	val, err = repl.re.Run("eth.mining")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, err = val.ToBoolean()
	if err != nil {
		t.Errorf("expected boolean, got %v", err)
	}
	if !mining {
		t.Errorf("expected true (mining), got false")
	}

	val, err = repl.re.Run("admin.startMining(4)")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if !mining {
		t.Errorf("expected true (mining), got false")
	}

	val, err = repl.re.Run("admin.stopMining()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if !mining {
		t.Errorf("expected true (mining), got false")
	}

}
