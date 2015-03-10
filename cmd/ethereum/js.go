// // Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
// //
// // This library is free software; you can redistribute it and/or
// // modify it under the terms of the GNU General Public
// // License as published by the Free Software Foundation; either
// // version 2.1 of the License, or (at your option) any later version.
// //
// // This library is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// // General Public License for more details.
// //
// // You should have received a copy of the GNU General Public License
// // along with this library; if not, write to the Free Software
// // Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// // MA 02110-1301  USA

package main

// import (
// 	"bufio"
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"os/signal"
// 	"path"
// 	"strings"

// 	"github.com/ethereum/go-ethereum/cmd/utils"
// 	// "github.com/ethereum/go-ethereum/core/types"
// 	"github.com/ethereum/go-ethereum/eth"
// 	"github.com/ethereum/go-ethereum/ethutil"
// 	"github.com/ethereum/go-ethereum/jsre"
// 	// "github.com/ethereum/go-ethereum/state"
// 	"github.com/ethereum/go-ethereum/xeth"
// 	"github.com/obscuren/otto"
// 	"github.com/peterh/liner"
// )

// func execJsFile(ethereum *eth.Ethereum, filename string) {
// 	re := jsre.New(assetPath)
// 	// re.Bind("eth", ...)

// 	if err := re.Load(filename); err != nil {
// 		utils.Fatalf("Javascript Error: %v", err)
// 	}
// }

// func runREPL(ethereum *eth.Ethereum) {
// 	re := jsre.New(assetPath)
// 	// xeth := xeth.New(ethereum, nil)
// 	jsre.RunREPL(path.Join(self.ethereum.DataDir, "repl.history"), re)
// }
