package gotry

type Func func() FuncReturn
type OnFuncErrorRetry func(retryCount int, returnValue interface{}, err error)
type Method func() error
type OnMethodErrorRetry func(retryCount int, err error)
type OnPanic func(panicError interface{})

type FuncReturn struct {
	ReturnValue interface{}
	Valid       bool
	Err         error
}

type Policy interface {
	SetRetry(retryLimit int) Policy
	SetInfiniteRetry(isInfinite bool) Policy
	SetRetryOnPanic(retryOnPanic bool) Policy
	ExecuteFunc(funcBody Func) FuncReturn
	ExecuteFuncWithRetryHook(funcBody Func, onRetry OnFuncErrorRetry, onPanic OnPanic) FuncReturn
	ExecuteMethod(methodBody Method) error
	ExecuteMethodWithRetryHook(methodBody Method, onErrorRetry OnMethodErrorRetry, onPanic OnPanic) error
}

type policy struct{
	retryOnPanic bool
	continueRetry func(int) bool
}

func NewPolicy() Policy {
	return &policy{
		retryOnPanic:  true,
		continueRetry: func(retryAttempt int) bool { return false },
	}
}

func (policy policy) SetRetry(retryLimit int) Policy{
	policy.continueRetry = func(retryAttempt int) bool {
		return retryAttempt < retryLimit
	}
	return &policy
}

func (policy policy) SetInfiniteRetry(isInfinite bool) Policy{
	policy.continueRetry = func(retryAttempt int) bool {
		return true
	}
	return &policy
}

func (policy policy) SetRetryOnPanic(retryOnPanic bool) Policy{
	policy.retryOnPanic = retryOnPanic
	return &policy
}

func (policy *policy) ExecuteFunc(funcBody Func) FuncReturn {
	return policy.ExecuteFuncWithRetryHook(funcBody, nil, nil)
}

func(policy *policy) ExecuteFuncWithRetryHook(funcBody Func, onRetry OnFuncErrorRetry, onPanic OnPanic) (funcReturn FuncReturn) {
	panicOccurred := false
	var wrappedOnPanic OnPanic = func(panicError interface{}){
		panicOccurred = true
		if onPanic != nil {
			onPanic(panicError)
		}
	}
	for i := 0; policy.continueRetry(retryAttempt(i)); i++ {
		panicOccurred = false
		var recoverableMethod = func() FuncReturn {
			defer func() {
				err := recover()
				if err != nil {
					wrappedOnPanic(err)
					panicIfExceedLimit(policy, i, err)
				}
			}()
			return funcBody()
		}
		funcReturn = recoverableMethod()
		if funcReturn.Err == nil && funcReturn.Valid && !panicOccurred {
			return
		}
		if onRetry != nil && !panicOccurred && policy.continueRetry(i) {
			onRetry(i+1, funcReturn.ReturnValue, funcReturn.Err)
		}
	}
	return
}

func retryAttempt(i int) int {
	return i - 1
}

func (policy *policy) ExecuteMethod(methodBody Method) error {
	return policy.ExecuteMethodWithRetryHook(methodBody, nil, nil)
}

func(policy *policy) ExecuteMethodWithRetryHook(methodBody Method, onErrorRetry OnMethodErrorRetry, onPanic OnPanic) error {
	function := func() FuncReturn{
		var err = methodBody()
		return FuncReturn{nil, true, err}
	}

	onFuncRetry := func(retryCount int, _ interface{}, err error){
		onErrorRetry(retryCount, err)
	}

	var funcReturn = policy.ExecuteFuncWithRetryHook(function, onFuncRetry, onPanic)
	return funcReturn.Err
}

func panicIfExceedLimit(policy *policy, i int, err interface{}) {
	if !(policy.retryOnPanic && policy.continueRetry(i)) {
		panic(err)
	}
}