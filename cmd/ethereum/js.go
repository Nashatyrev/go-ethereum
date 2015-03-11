// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/jeth"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

func jethre(ethereum *eth.Ethereum) *ethutil.REPL {
	re := jsre.New(assetPath)
	repl := NewREPL(re)
	// extend the repl to provide a console UI for xeth
	frontend := consoleFrontend{ethereum, repl}

	// js - xeth binding happens here
	xeth := xeth.New(ethereum, frontend)
	ethApi := rpc.NewEthereumApi(xeth, ethereum.DataDir)
	re.Bind("jeth", jeth.New(ethApi, re.ToVal))
	re.Bind("eth", &ethadmin{ethereum, xeth, re.ToVal})

	err := re.Load(jsre.BigNumber_JS)

	if err != nil {
		utils.Fatalf("Error loading bignumber.js: %v", err)
	}

	// we need to declare a dummy setTimeout. Otto does not support it
	_, err = re.Run("setTimeout = function(cb, delay) {};")
	if err != nil {
		utils.Fatalf("Error defining setTimeout: %v", err)
	}

	_, err = re.Run(jsre.Ethereum_JS)
	if err != nil {
		utils.Fatalf("Error loading ethereum.js: %v", err)
	}

	_, err = re.Run("var web3 = require('web3');")
	if err != nil {
		utils.Fatalf("Error requiring web3: %v", err)
	}

	_, err = re.Run("web3.setProvider(jeth)")
	if err != nil {
		utils.Fatalf("Error setting web3 provider: %v", err)
	}
	return repl
}

// consoleFrontend provides the UI callback for xeth
// for account unlocking and transaction confirmation
// we just wrap the same REPL/prompter that is used by the console
type consoleFrontend struct {
	ethereum *eth.Ethereum
	*ethutil.REPL
}

func (self consoleFrontend) ConfirmTransaction(tx *types.Transaction) bool {
	p := fmt.Sprintf("Confirm Transaction %v\n[y/n] ", tx)
	answer, _ := self.Prompt(p)
	return strings.HasPrefix(strings.Trim(answer, " "), "y")
}

func (self consoleFrontend) UnlockAccount(addr []byte) bool {
	fmt.Printf("Please unlock account %x.\n", addr)
	pass, err := self.PasswordPrompt("Passphrase: ")
	if err != nil {
		return false
	}
	// TODO: allow retry
	if err := self.ethereum.AccountManager().Unlock(addr, pass); err != nil {
		return false
	} else {
		fmt.Println("Account is now unlocked for this session.")
		return true
	}
}

// the ethereum client admin task bindings for js
type ethadmin struct {
	eth   *eth.Ethereum
	xeth  *xeth.XEth
	toVal func(interface{}) otto.Value
}

func (self *ethadmin) IsMining(call otto.FunctionCall) otto.Value {
	return self.toVal(self.xeth.IsMining())
}

func (self *ethadmin) SetMining(call otto.FunctionCall) otto.Value {
	shouldmine, err := call.Argument(0).ToBoolean()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	mining := self.xeth.SetMining(shouldmine)
	return self.toVal(mining)
}

func (self *ethadmin) SuggestPeer(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	if err := self.eth.SuggestPeer(nodeURL); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *ethadmin) Import(call otto.FunctionCall) otto.Value {
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

	self.eth.ChainManager().Reset()
	if err = self.eth.ChainManager().InsertChain(blocks); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (self *ethadmin) Export(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	data := self.eth.ChainManager().Export()
	if err := ethutil.WriteFile(fn, data); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (self *ethadmin) DumpBlock(call otto.FunctionCall) otto.Value {
	var block *types.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = self.eth.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = self.eth.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

	} else {
		block = self.eth.ChainManager().CurrentBlock()
		block = self.eth.ChainManager().CurrentBlock()
	}
	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	statedb := state.New(block.Root(), self.eth.StateDb())
	dump := statedb.RawDump()
	return self.toVal(dump)

}
