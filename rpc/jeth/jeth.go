package jeth

import (
	"encoding/json"
	"github.com/obscuren/otto"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	jsonrpcver       = "2.0"
	maxSizeReqLength = 1024 * 1024 // 1MB
)

var jlogger = logger.NewLogger("RPC-JEth")

type Jeth struct {
	ethApi *rpc.EthereumApi
	toVal  func(interface{}) otto.Value
}

func New(ethApi *rpc.EthereumApi, toVal func(interface{}) otto.Value) *Jeth {
	return &Jeth{ethApi, toVal}
}

func (self *Jeth) err(code int, msg string, id interface{}) otto.Value {
	jsonerr := &rpc.RpcErrorObject{code, msg}
	response := &rpc.RpcErrorResponse{JsonRpc: jsonrpcver, ID: id, Error: jsonerr}
	result, _ := json.Marshal(response)
	return self.toVal(result)
}

func (self *Jeth) Send(call otto.FunctionCall) (ottoResponse otto.Value) {
	jlogger.DebugDetailln("Handling request")
	jsonreq, err := call.Argument(0).ToString()
	if err != nil {
		return self.err(-32700, err.Error(), nil)
	}
	if len(jsonreq) > maxSizeReqLength {
		return self.err(-32700, "Error: Request too large", nil)
	}
	var req rpc.RpcRequest
	err = json.Unmarshal([]byte(jsonreq), &req)
	if err != nil {
		return self.err(-32700, err.Error(), nil)
	}

	var response interface{}
	err = self.ethApi.GetRequestReply(&req, &response)
	if err != nil {
		return self.err(-32603, err.Error(), req.ID)
	}
	rpcresponse := &rpc.RpcSuccessResponse{JsonRpc: jsonrpcver, ID: req.ID, Result: response}
	result, _ := json.Marshal(rpcresponse)
	return self.toVal(result)
}
