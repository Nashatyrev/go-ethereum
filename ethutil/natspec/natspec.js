/*
    This file is part of natspec.js.

    natspec.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    natspec.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with natspec.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file natspec.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

//var abi = require('./node_modules/ethereum.js/lib/abi.js'); 
//var abi = require('./lib/abi.js'); 

/**
 * This object should be used to evaluate natspec expression
 * It has one method evaluateExpression which shoul be used
 */
var natspec = (function () {
    /// Helper method
    /// Should be called to copy values from object to global context
    var copyToContext = function (obj, context) {
        var keys = Object.keys(obj);
        keys.forEach(function (key) {
            context[key] = obj[key];
        });
    }

    /// this function will not be used in 'production' natspec evaluation
    /// it's only used to enable tests in node environment
    /// it copies all functions from current context to nodejs global context
    var copyToNodeGlobal = function (obj) {
        if (typeof global === 'undefined') {
            return;
        }
        copyToContext(obj, global);
    };

    /// Helper method
    /// Should be called to get method with given name from the abi
    /// @param contract's abi
    /// @param name of the method that we are looking for
    var getMethodWithName = function(abi, name) {
        return abi.filter(function (method) {
            return method.name === name;
        })[0];
    };

    /// Function called to get all contract's storage values
    /// @returns hashmap with contract properties which are used
    /// TODO: check if this function will be used
    var getContractProperties = function (address, abi) {
        return {};
    };

    /// Function called to get all contract method input variables
    /// @returns hashmap with all contract's method input variables
    var getMethodInputParams = function (method, transaction) {
        // do it with output formatter (cause we have to decode)
        var params = abi.formatOutput(method.inputs, '0x' + transaction.params[0].data.slice(10)); 

        return method.inputs.reduce(function (acc, current, index) {
            acc[current.name] = params[index];
            return acc;
        }, {});
    };
    
    /// Should be called to evaluate single expression
    /// Is internally using javascript's 'eval' method
    /// @param expression which should be evaluated
    /// @param [call] object containing contract abi, transaction, called method
    /// TODO: separate evaluation from getting input params, so as not to spoil 'evaluateExpression' function
    var evaluateExpression = function (expression, call) {
        var self = this;
        
        if (!!call) {
            try {
                var method = getMethodWithName(call.abi, call.method);
                var params = getMethodInputParams(method, call.transaction); 
                copyToContext(params, self);
            }
            catch (err) {
                return err;//"Natspec evaluation failed, wrong input params";
            }
        }

        // used only for tests
        copyToNodeGlobal(self);

        var evaluatedExpression = "";

        // match everything in `` quotes
        var pattern = /\`(?:\\.|[^`\\])*\`/gim
        var match;
        var lastIndex = 0;
        while ((match = pattern.exec(expression)) !== null) {
            var startIndex = pattern.lastIndex - match[0].length;

            var toEval = match[0].slice(1, match[0].length - 1);

            evaluatedExpression += expression.slice(lastIndex, startIndex);

            var evaluatedPart;
            try {
                evaluatedPart = eval(toEval).toString(); 
            }
            catch (err) {
                evaluatedPart = 'undefined'; 
            }

            evaluatedExpression += evaluatedPart;
            lastIndex = pattern.lastIndex;
        }

        evaluatedExpression += expression.slice(lastIndex);
        
        return evaluatedExpression;
    };

    return {
        evaluateExpression: evaluateExpression
    };

})();

//module.exports = natspec; 

