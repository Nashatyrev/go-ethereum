package jeth

import (
	"encoding/json"
	"fmt"
	"github.com/obscuren/otto"

	"github.com/ethereum/go-ethereum/rpc"
)

const (
	jsonrpcver       = "2.0"
	maxSizeReqLength = 1024 * 1024 // 1MB
)

type Jeth struct {
	ethApi *rpc.EthereumApi
	toVal  func(interface{}) otto.Value
}

func New(ethApi *rpc.EthereumApi, toVal func(interface{}) otto.Value) *Jeth {
	return &Jeth{ethApi, toVal}
}

func (self *Jeth) err(code int, msg string, id interface{}) otto.Value {
	rpcerr := &rpc.RpcErrorObject{code, msg}
	rpcresponse := &rpc.RpcErrorResponse{Jsonrpc: jsonrpcver, Id: id, Error: rpcerr}
	// rpcresponse := &rpc.RpcErrorResponse{JsonRpc: jsonrpcver, ID: id, Error: rpcerr}
	return self.toVal(rpcresponse)
}

func (self *Jeth) Send(call otto.FunctionCall) (response otto.Value) {
	reqif, err := call.Argument(0).Export()
	if err != nil {
		return self.err(-32700, err.Error(), nil)
	}
	fmt.Printf("reqif: %#v\n", reqif)

	jsonreq, err := json.Marshal(reqif)

	var req rpc.RpcRequest
	err = json.Unmarshal(jsonreq, &req)
	fmt.Printf("req: %#v\n", req)

	var respif interface{}
	err = self.ethApi.GetRequestReply(&req, &respif)
	if err != nil {
		return self.err(-32603, err.Error(), req.Id)
	}
	rpcresponse := &rpc.RpcSuccessResponse{Jsonrpc: jsonrpcver, Id: req.Id, Result: respif}
	// rpcresponse := &rpc.RpcSuccessResponse{JsonRpc: jsonrpcver, ID: req.ID, Result: respif}
	fmt.Printf("rpcresponse: %#v\n", rpcresponse)
	response = self.toVal(rpcresponse)
	fmt.Printf("response: %#v\n", response)
	return
}
