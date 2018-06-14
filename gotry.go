package gotry

type Func func() (returnValue interface{}, isReturnValueValid bool, err error)
type OnFuncErrorRetry func(retryCount int, returnValue interface{}, err error)
type Method func() error
type OnMethodErrorRetry func(retryCount int, err error)
type OnPanic func(panicError interface{})

type Policy struct{
	retryLimit int
	isInfinite bool
}

func (policy *Policy)continueRetry(i int) bool {
	return i < policy.retryLimit || policy.isInfinite
}

func (policy *Policy) SetRetry(retryLimit int) {
	policy.retryLimit = retryLimit
}

func (policy *Policy) SetInfiniteRetry(isInfinite bool) {
	policy.isInfinite = isInfinite
}

func (policy *Policy) ExecuteFunc(funcBody Func) (returnValue interface{}, isReturnValueValid bool, err error) {
	return policy.ExecuteFuncWithRetryHook(funcBody, nil, nil)
}

func(policy *Policy) ExecuteFuncWithRetryHook(funcBody Func, onRetry OnFuncErrorRetry, onPanic OnPanic) (returnValue interface{}, isReturnValueValid bool, err error) {
	panicOccurred := false
	var wrappedOnPanic OnPanic = func(panicError interface{}){
		panicOccurred = true
		if onPanic != nil {
			onPanic(panicError)
		}
	}
	for i := 0; policy.continueRetry(i); i++ {
		panicOccurred = false
		var recoverableMethod = func() (returnValue interface{}, isReturnValueValid bool, err error) {
			defer func() {
				err := recover()
				if err != nil {
					wrappedOnPanic(err)
					panicIfExceedLimit(policy, i, err)
				}
			}()
			return funcBody()
		}
		returnValue, isReturnValueValid, err = recoverableMethod()
		if err == nil && isReturnValueValid && !panicOccurred {
			return returnValue, true, nil
		}
		if onRetry != nil && !panicOccurred {
			onRetry(i+1, returnValue, err)
		}
	}
	return
}

func (policy *Policy) ExecuteMethod(methodBody Method) error {
	return policy.ExecuteMethodWithRetryHook(methodBody, nil, nil)
}

func(policy *Policy) ExecuteMethodWithRetryHook(methodBody Method, onErrorRetry OnMethodErrorRetry, onPanic OnPanic) error {
	function := func() (interface{}, bool, error){
		var err = methodBody()
		return nil, true, err
	}

	onFuncRetry := func(retryCount int, _ interface{}, err error){
		onErrorRetry(retryCount, err)
	}

	var _, _, funcError = policy.ExecuteFuncWithRetryHook(function, onFuncRetry, onPanic)
	return funcError
}

func panicIfExceedLimit(policy *Policy, i int, err interface{}) {
	if !policy.continueRetry(i) {
		panic(err)
	}
}