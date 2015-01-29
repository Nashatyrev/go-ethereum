package natspec

import (
	"encoding/json"
	// "fmt"
	"github.com/ethereum/go-ethereum/javascript"
)

type ptype int

/*
NatSpec processes contract metadata
initialised from JSON-format contract definitions

 def: contract ABI definition
    - https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI#json
 doc: contract user documentation
    - https://github.com/ethereum/wiki/wiki/Ethereum-Natural-Specification-Format#user-documentation
 def: contract ABI definition
    - https://github.com/ethereum/wiki/wiki/Ethereum-Natural-Specification-Format#developer-documentation
*/
type NatSpec struct {
	def     [](map[string]interface{})
	doc     map[string]interface{}
	dev     map[string]interface{}
	methods map[string]*meth
	jsre    *javascript.JSRE
}

// This can be cached and called multiple times when the method is called with different parameters.
func New(defJson, docJson []byte, jsreF func() (*javascript.JSRE, error)) (self *NatSpec, err error) {
	def := []map[string](interface{}){}
	if err = json.Unmarshal(defJson, &def); err != nil {
		return
	}

	doc := make(map[string]interface{})
	if err = json.Unmarshal(docJson, &doc); err != nil {
		return
	}
	self = &NatSpec{def, doc, nil, make(map[string]*meth), nil}
	if self.jsre, err = jsreF(); err != nil {
		return
	}
	return
}

type meth struct {
	args  []string
	types []ptype
	// signature string
	// key       string
	notice string
}

// var m = &method{}

func (self *NatSpec) Notice(method string, params ...interface{}) (notice string, err error) {
	m := self.methods[method]
	if m == nil {
		var info, param map[string]interface{}
		for _, info = range self.def {
			name, ok := info["name"].(string)
			if !ok {
				return
			}
			if name == method {
				break
			}
		}
		var args []string
		var types []ptype
		inputs, ok := info["inputs"].([]map[string](interface{}))
		if !ok {
			return
		}
		var name string
		var typ ptype
		for _, param = range inputs {
			name, ok = param["name"].(string)
			args = append(args, name)
			typ, ok = param["type"].(ptype)
			types = append(types, typ)
		}
		m = &meth{args, types /* signature, key, */, notice}
		self.methods[method] = m
	}
	// key := crypto.Sha3(signature)
	key := ""
	methods, ok := self.doc["methods"].(map[string](map[string](interface{})))
	if !ok {
		return
	}
	meth := methods[key]
	// meth, ok := methods[key].(map[string](interface{}))
	if !ok {
		return
	}
	expression, ok := meth["notice"].(string)
	notice = expression
	// notice = "" // evaluate(expression)
	return
}

var evalScript = "// match everything in `` quotes" +
	"var pattern = /\\\\`(?:\\\\.|[^`\\\\])*\\\\`/gim" +
	`var match;
var lastIndex = 0;
while ((match = pattern.exec(expression)) !== null) {
  var startIndex = pattern.lastIndex - match[0].length;
  var toEval = match[0].slice(1, match[0].length - 1);
  evaluatedExpression += expression.slice(lastIndex, startIndex);
  evaluatedExpression += eval(toEval).toString();
  lastIndex = pattern.lastIndex;
}
evaluatedExpression += expression.slice(lastIndex);
return evaluatedExpression;
`

	// @title: This is a title that should describe the contract and go above the contract definition
	// @author: The name of the author of the contract. Should also go above the contract definition.
	// @notice: Represents user documentation. This is the text that will appear to the user to notify him of what the function he is about to execute is doing
	// @dev: Represents developer documentation. This is documentation that would only be visible to the developer.
	// @param: Documents a parameter just like in doxygen. Has to be followed by the parameter name.
	// @return: Documents the return type of a contract's function.
