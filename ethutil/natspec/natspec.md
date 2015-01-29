In the ccp client natspec is not really modularized. When authenticating a transaction (before sending), the GUI backend retrieves the user notice as a string and evaluates it with the transaction data. Retrieving the notice is done by NatSpecHandler::getUserNotice  https://github.com/ethereum/cpp-ethereum/blob/develop/alethzero/NatspecHandler.cpp#L66 

First it retrieves the user documentation JSON for the current contract _d_<sub>_c_</sub> from local db storage https://github.com/ethereum/cpp-ethereum/blob/develop/alethzero/NatspecHandler.cpp#L58 then consults the transaction data (in ABI format) to extract the method key _k_<sub>_m_</sub> (first bytes of the hash of the function signature). Then it returns _d_<sub>_c_</sub>[_k_<sub>_m_</sub>]["notice"]. (Btw, since transaction data is available, evaluation could and should happen here too).

The evaluation of embedded expressions is performed by evaluating the notice string in a JS runtime with preloaded helper `natspec.js` . https://github.com/ethereum/cpp-ethereum/blob/develop/alethzero/OurWebThreeStubServer.cpp#L101

This js file (https://github.com/ethereum/cpp-ethereum/blob/develop/libjsqrc/natspec.js) takes care of setting up the evaluation context to interpret the parameters within the evaluated expression (locally scoped variables in js) https://github.com/ethereum/cpp-ethereum/blob/develop/libjsqrc/natspec.js#L16. The context comes from the `web3` js interface, https://github.com/ethereum/cpp-ethereum/blob/develop/libjsqrc/natspec.js#L48

    var abi = web3._currentContractAbi;
    var address = web3._currentContractAddress;
    var methodName = web3._currentContractMethodName;
    var params = web3._currentContractMethodParams;

Based on the last line, actual input values of transaction data can be assigned to the correct parameters https://github.com/ethereum/cpp-ethereum/blob/develop/libjsqrc/natspec.js#L34 


## Issues 

**Issue**
Relying on web3 for eval context is limiting since it assumes all transactions are sent via a js DAPP, but we want to generate these notices irrespective of the environment the transaction is called from. Also arbitrary code execution in a rich environment has security risks.

**Solution**: not to rely on web3, but let the backend assign actual values to method params taken directly from the transaction parameters. Use a bare js environment.

**Issue**
Relying on web3 only for eval context is limiting in another way as well. By itself since it does not provide bindings for important variables like `message.caller`. 

**Solution**: implement these bindings explicitly in natspec handler. Make it explicit in the spec which bindings are supported. Ideally the full suite of solidity special values, see  https://github.com/ethereum/wiki/wiki/Solidity-Tutorial

**Issue**
Relying on web3 to provide values for the current transaction parameters (`web3._currentContractMethodParams`) is problematic. It is possible that when the backend generates the confirmation of the transaction with params, the confirmation message will use a different set. This is possible since an async js process in the DAPP could change values after the request is relayed to the backend but before the natspec notice expression is evaled. This may easily result in the user confirming one thing while sending another thing. Even if the contract is audited. 

**Solution**: implement transaction data param bindings explicitly in natspec handler using the very values that are passed.

**Issue**
When the transaction is being sent, the method name is already parsed, so no need for natspec to parse it again. If NatSpec only has access to ABI-formatted transaction data, then it has to iterate through the json structure sha3 the key and compare to the transaction function. This is rather inefficient. https://github.com/ethereum/cpp-ethereum/blob/develop/alethzero/NatspecHandler.cpp#L75

**Solution**
NatSpec user notice method receives parsed transaction params.

**Issue**
NatSpec notices may be needed outside Mist.

**Solution** 
Separate package or util

**Issue** 
Depending on the context, the contract code and or the natspec JSON definitions can be stored and retrieved in a various ways at the discretion of the caller. This is not possible if definition retrieval is handled within NatSpec.

**Solution**
The caller passes the contract definition and user documentation to NatSpec.
These JSON string are either retrieved from some storage (swarm down the line) or - if we are inside an IDE - we assume to have access to the solidity lib and can generate new JSONs for new/modified/updated contracts.
The contract definition could be passed to natspec in processed internal format https://github.com/ethereum/go-ethereum/blob/develop/accounts/abi/abi.go

The user documentation (JSON string) for the contract is provided by solidity
https://github.com/ethereum/cpp-ethereum/blob/develop/libsolidity/InterfaceHandler.h#L82

The ABI interface definition (JSON string) for the contract is provided by solidity:
https://github.com/ethereum/cpp-ethereum/blob/develop/libsolidity/InterfaceHandler.h#L76

# API
// This can be cached and called multiple times when the method is called with different parameters.

``` go
    natSpec, err := natspec.New(abiDefJSON, userDocJSON)
    s := natSpec.Notice(method, t0, t1, t2)
```

# Resources:

- https://github.com/ethereum/wiki/wiki/Ethereum-Natural-Specification-Format
- https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI
- https://github.com/ethereum/wiki/wiki/JavaScript-API

