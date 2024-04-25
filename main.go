package main

import (
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"test2": run.InitializedTestCaseFn(Test2),
	"test1": run.InitializedTestCaseFn(Test1),
}

func main() {
	run.InvokeMap(testcases)
}
