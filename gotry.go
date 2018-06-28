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
	SetOnFuncRetry(onRetry OnFuncErrorRetry) Policy
	SetOnMethodRetry(onRetry OnMethodErrorRetry) Policy
	SetOnPanic(onPanic OnPanic) Policy
	ExecuteFunc(funcBody Func) FuncReturn
	ExecuteMethod(methodBody Method) error
}

type policy struct{
	retryOnPanic bool
	continueRetry func(int) bool
	onFuncRetry OnFuncErrorRetry
	onMethodRetry OnMethodErrorRetry
	onPanic OnPanic
}

func NewPolicy() Policy {
	return &policy{
		retryOnPanic:  true,
		continueRetry: func(retryAttempt int) bool { return false },
	}
}

func (policy policy) SetRetry(retryLimit int) Policy{
	policy.continueRetry = func(retryAttempt int) bool {
		return retryAttempt <= retryLimit
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

func (policy policy) SetOnFuncRetry(onRetry OnFuncErrorRetry) Policy{
	policy.onFuncRetry = onRetry
	return &policy
}

func (policy policy) SetOnMethodRetry(onRetry OnMethodErrorRetry) Policy{
	policy.onMethodRetry = onRetry
	return &policy
}

func (policy policy) SetOnPanic(onPanic OnPanic) Policy{
	policy.onPanic = onPanic
	return &policy
}

func(policy *policy) ExecuteFunc(funcBody Func) (funcReturn FuncReturn) {
	panicOccurred := false
	var notifyPanic OnPanic = func(panicError interface{}){
		panicOccurred = true
		if policy.onPanic != nil {
			policy.onPanic(panicError)
		}
	}
	for i := 0; policy.continueRetry(i); i++ {
		panicOccurred = false
		var recoverableMethod = func() FuncReturn {
			defer func() {
				panicErr := recover()
				if panicErr != nil {
					notifyPanic(panicErr)
					panicIfExceedLimit(policy, retryAttempt(i), panicErr)
				}
			}()
			return funcBody()
		}
		funcReturn = recoverableMethod()
		if funcReturn.Err == nil && funcReturn.Valid && !panicOccurred {
			return
		}
		if policy.onFuncRetry != nil && !panicOccurred && policy.continueRetry(retryAttempt(i)) {
			policy.onFuncRetry(retryAttempt(i), funcReturn.ReturnValue, funcReturn.Err)
		}
	}
	return
}

func retryAttempt(i int) int {
	return i+1
}

func(policy *policy) ExecuteMethod(methodBody Method) error {
	function := func() FuncReturn{
		var err = methodBody()
		return FuncReturn{nil, true, err}
	}

	wrappedPolicy := policy.SetOnFuncRetry(func(retryCount int, _ interface{}, err error){
		policy.onMethodRetry(retryCount, err)
	})

	var funcReturn = wrappedPolicy.ExecuteFunc(function)
	return funcReturn.Err
}

func panicIfExceedLimit(policy *policy, i int, err interface{}) {
	if !(policy.retryOnPanic && policy.continueRetry(i)) {
		panic(err)
	}
}