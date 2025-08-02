// Test script for debugging features

// Global variables
var globalCounter = 0;
var globalMessage = "Hello from global scope";

// Simple function with local variables
function calculateSum(a, b) {
    var localSum = a + b;
    var localMessage = "Calculating sum...";
    
    globalCounter++;
    
    return localSum;
}

// Function with closure
function createMultiplier(factor) {
    var closureFactor = factor;
    
    return function(num) {
        var result = num * closureFactor;
        console.log("Multiplying", num, "by", closureFactor, "=", result);
        return result;
    };
}

// Native function usage
function testNativeFunctions() {
    var arr = [1, 2, 3, 4, 5];
    
    // Using native Array methods
    var doubled = arr.map(function(x) {
        return x * 2;
    });
    
    var sum = arr.reduce(function(acc, val) {
        return acc + val;
    }, 0);
    
    console.log("Original array:", arr);
    console.log("Doubled array:", doubled);
    console.log("Sum:", sum);
    
    // Using native Math functions
    var max = Math.max(10, 5, 20, 15);
    var sqrt = Math.sqrt(16);
    
    console.log("Max value:", max);
    console.log("Square root of 16:", sqrt);
    
    return {
        array: arr,
        doubled: doubled,
        sum: sum,
        max: max,
        sqrt: sqrt
    };
}

// Complex object with nested properties
function createComplexObject() {
    var obj = {
        name: "Debug Test",
        version: 1.0,
        nested: {
            level1: {
                level2: {
                    value: "Deep nested value"
                }
            }
        },
        array: [1, 2, 3, { inner: "array object" }],
        method: function() {
            return "Method result";
        }
    };
    
    return obj;
}

// Control flow test
function testControlFlow(n) {
    var result = "";
    
    if (n > 10) {
        result = "Greater than 10";
    } else if (n > 5) {
        result = "Between 6 and 10";
    } else {
        result = "5 or less";
    }
    
    // Loop test
    var loopResult = 0;
    for (var i = 0; i < n; i++) {
        loopResult += i;
    }
    
    return {
        comparison: result,
        sum: loopResult
    };
}

// Exception handling test
function testExceptionHandling() {
    try {
        var x = 10;
        var y = 0;
        
        if (y === 0) {
            throw new Error("Division by zero!");
        }
        
        return x / y;
    } catch (e) {
        console.log("Caught error:", e.message);
        return -1;
    } finally {
        console.log("Finally block executed");
    }
}

// Main execution
console.log("=== Starting Debug Test ===");

var sum1 = calculateSum(5, 3);
console.log("Sum of 5 + 3 =", sum1);

var sum2 = calculateSum(10, 20);
console.log("Sum of 10 + 20 =", sum2);

var multiplier = createMultiplier(5);
var result1 = multiplier(10);
var result2 = multiplier(7);

var nativeResults = testNativeFunctions();
console.log("Native function results:", nativeResults);

var complexObj = createComplexObject();
console.log("Complex object created:", complexObj.name);

var flowResult = testControlFlow(7);
console.log("Control flow result:", flowResult);

var errorResult = testExceptionHandling();
console.log("Exception handling result:", errorResult);

console.log("Global counter final value:", globalCounter);
console.log("=== Debug Test Complete ===");