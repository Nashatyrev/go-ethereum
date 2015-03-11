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
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/jsre"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/jeth"
	"github.com/ethereum/go-ethereum/xeth"
)

func jethre(ethereum *eth.Ethereum) *ethutil.REPL {
	re := jsre.New(assetPath)
	repl := ethutil.NewREPL(re)
	// extend the repl to provide a console UI for xeth
	frontend := consoleFrontend{ethereum, repl}

	// js - xeth binding happens here
	xeth := xeth.New(ethereum, frontend)
	ethApi := rpc.NewEthereumApi(xeth, ethereum.DataDir)
	re.Bind("jeth", jeth.New(ethApi, re.ToVal))

	err := re.Load("bignumber.min.js")

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

// func (self *jsre) initStdFuncs() {
// 	t, _ := self.re.Vm.Get("eth")
// 	eth := t.Object()
// 	eth.Set("connect", self.connect)
// 	eth.Set("stopMining", self.stopMining)
// 	eth.Set("startMining", self.startMining)
// 	eth.Set("dump", self.dump)
// 	eth.Set("export", self.export)
// }

// /*
//  * The following methods are natively implemented javascript functions.
//  */

// func (self *jsre) dump(call otto.FunctionCall) otto.Value {
// 	var block *types.Block

// 	if len(call.ArgumentList) > 0 {
// 		if call.Argument(0).IsNumber() {
// 			num, _ := call.Argument(0).ToInteger()
// 			block = self.ethereum.ChainManager().GetBlockByNumber(uint64(num))
// 		} else if call.Argument(0).IsString() {
// 			hash, _ := call.Argument(0).ToString()
// 			block = self.ethereum.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
// 		} else {
// 			fmt.Println("invalid argument for dump. Either hex string or number")
// 		}

// 		if block == nil {
// 			fmt.Println("block not found")

// 			return otto.UndefinedValue()
// 		}

// 	} else {
// 		block = self.ethereum.ChainManager().CurrentBlock()
// 	}

// 	statedb := state.New(block.Root(), self.ethereum.StateDb())

// 	v, _ := self.re.Vm.ToValue(statedb.RawDump())

// 	return v
// }

// func (self *jsre) stopMining(call otto.FunctionCall) otto.Value {
// 	self.xeth.Miner().Stop()
// 	return otto.TrueValue()
// }

// func (self *jsre) startMining(call otto.FunctionCall) otto.Value {
// 	self.xeth.Miner().Start()
// 	return otto.TrueValue()
// }

// func (self *jsre) connect(call otto.FunctionCall) otto.Value {
// 	nodeURL, err := call.Argument(0).ToString()
// 	if err != nil {
// 		return otto.FalseValue()
// 	}
// 	if err := self.ethereum.SuggestPeer(nodeURL); err != nil {
// 		return otto.FalseValue()
// 	}
// 	return otto.TrueValue()
// }

// func (self *jsre) export(call otto.FunctionCall) otto.Value {
// 	if len(call.ArgumentList) == 0 {
// 		fmt.Println("err: require file name")
// 		return otto.FalseValue()
// 	}

// 	fn, err := call.Argument(0).ToString()
// 	if err != nil {
// 		fmt.Println(err)
// 		return otto.FalseValue()
// 	}

// 	data := self.ethereum.ChainManager().Export()

// 	if err := ethutil.WriteFile(fn, data); err != nil {
// 		fmt.Println(err)
// 		return otto.FalseValue()
// 	}

// 	return otto.TrueValue()
// }
