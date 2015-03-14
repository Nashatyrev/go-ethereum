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

var port = 30300

func testJEthRE(t *testing.T) (repl *jsre, ethereum *eth.Ethereum, err error) {
	os.RemoveAll("/tmp/eth/")
	err = os.MkdirAll("/tmp/eth/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/", os.ModePerm)
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

	port++
	ethereum, err = eth.New(&eth.Config{
		DataDir:        "/tmp/eth",
		AccountManager: accounts.NewManager(ks),
		Port:           fmt.Sprintf("%d", port),
		MaxPeers:       10,
		Name:           "test",
	})

	if err != nil {
		t.Errorf("%v", err)
		return
	}
	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	repl = newJSRE(ethereum, assetPath)
	return
}

func TestNodeInfo(t *testing.T) {
	repl, ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	val, err := repl.re.Run("admin.nodeInfo()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	exp, err := val.Export()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	nodeInfo, ok := exp.(*eth.NodeInfo)
	if !ok {
		t.Errorf("expected nodeInfo, got %v", err)
	}
	exp = "test"
	got := nodeInfo.Name
	if exp != got {
		t.Errorf("expected %v, got %v", exp, got)
	}
	exp = 30301
	port := nodeInfo.DiscPort
	if exp != port {
		t.Errorf("expected %v, got %v", exp, port)
	}
	exp = 30301
	port = nodeInfo.TCPPort
	if exp != port {
		t.Errorf("expected %v, got %v", exp, port)
	}
}

func TestAccounts(t *testing.T) {
	repl, ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

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
	addrslice, err := val.Export()
	if err != nil {
		t.Errorf("expected string, got %v", err)
	}
	addr, ok := addrslice.([]byte)
	if !ok {
		t.Errorf("expected []byte, got %v", err)
	}
	fmt.Printf("addr: %x", addr)

	val, err = repl.re.Run("eth.accounts")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	exp, err := val.Export()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addrs, ok := exp.([]string)
	if !ok {
		t.Errorf("expected []string, got %v", err)
	}
	if len(addrs) != 2 || (ethutil.Bytes2Hex(addr) != addrs[0][2:] && ethutil.Bytes2Hex(addr) != addrs[1][2:]) {
		t.Errorf("expected addrs == [<default>, <new>], got %v (%v)", addrs, ethutil.Bytes2Hex(addr))
	}

}

func TestBlockChain(t *testing.T) {
	repl, ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	// should get current block
	val0, err := repl.re.Run("admin.dumpBlock()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	fn := "/tmp/eth/data/blockchain.0"
	_, err = repl.re.Run("admin.export(\"" + fn + "\")")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if _, err = os.Stat(fn); err != nil {
		t.Errorf("expected no error on file, got %v", err)
	}

	_, err = repl.re.Run("admin.import(\"" + fn + "\")")
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
}

func TestMining(t *testing.T) {
	repl, ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	val, err := repl.re.Run("eth.mining")
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

func TestRPC(t *testing.T) {
	repl, ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	val, err := repl.re.Run(`admin.startRPC("127.0.0.1", 5004)`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	success, _ := val.ToBoolean()
	if !success {
		t.Errorf("expected true (started), got false")
	}
}
