package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
	"github.com/obscuren/otto"
)

/*
node admin bindings
*/

func (js *jsre) adminBindings() {
	js.re.Set("admin", struct{}{})
	t, _ := js.re.Get("admin")
	admin := t.Object()
	admin.Set("addPeer", js.addPeer)
	admin.Set("setMining", js.setMining)
	admin.Set("import", js.importChain)
	admin.Set("export", js.exportChain)
	admin.Set("dumpBlock", js.dumpBlock)
}

func (js *jsre) setMining(call otto.FunctionCall) otto.Value {
	shouldmine, err := call.Argument(0).ToBoolean()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	mining := js.xeth.SetMining(shouldmine)
	return js.re.ToVal(mining)
}

func (js *jsre) addPeer(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	if err := js.ethereum.SuggestPeer(nodeURL); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) importChain(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	var fh *os.File
	fh, err = os.OpenFile(fn, os.O_RDONLY, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	defer fh.Close()

	var blocks types.Blocks
	if err = rlp.Decode(fh, &blocks); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	js.ethereum.ChainManager().Reset()
	if err = js.ethereum.ChainManager().InsertChain(blocks); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (js *jsre) exportChain(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	data := js.ethereum.ChainManager().Export()
	if err := ethutil.WriteFile(fn, data); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (js *jsre) dumpBlock(call otto.FunctionCall) otto.Value {
	var block *types.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = js.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = js.ethereum.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

	} else {
		block = js.ethereum.ChainManager().CurrentBlock()
	}
	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	statedb := state.New(block.Root(), js.ethereum.StateDb())
	dump := statedb.RawDump()
	return js.re.ToVal(dump)

}
