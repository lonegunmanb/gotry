package gotry

type Func func() FuncReturn
type OnFuncError func(retriedCount int, returnValue interface{}, err error)
type Method func() error
type OnMethodError func(retriedCount int, err error)
type OnPanic func(panicError interface{})

type FuncReturn struct {
	ReturnValue interface{}
	Valid       bool
	Err         error
}

type Policy interface {
	SetRetry(retryLimit int) Policy
	SetInfiniteRetry() Policy
	SetRetryPredicate(predicate func()bool) Policy
	SetRetryOnPanic(retryOnPanic bool) Policy
	SetOnFuncRetry(onRetry OnFuncError) Policy
	SetOnMethodRetry(onRetry OnMethodError) Policy
	SetOnPanic(onPanic OnPanic) Policy
	ExecuteFunc(funcBody Func) FuncReturn
	ExecuteMethod(methodBody Method) error
}

type policy struct{
	retryOnPanic  bool
	continueRetry func(int) bool
	onFuncError   OnFuncError
	onMethodError OnMethodError
	onPanic       OnPanic
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

func (policy policy) SetInfiniteRetry() Policy {
	policy.continueRetry = func(retryAttempt int) bool {
		return true
	}
	return &policy
}

func (policy policy) SetRetryPredicate(predicate func()bool) Policy{
	policy.continueRetry = func(int)bool {
		return predicate()
	}
	return &policy
}

func (policy policy) SetRetryOnPanic(retryOnPanic bool) Policy{
	policy.retryOnPanic = retryOnPanic
	return &policy
}

func (policy policy) SetOnFuncRetry(onRetry OnFuncError) Policy{
	policy.onFuncError = onRetry
	return &policy
}

func (policy policy) SetOnMethodRetry(onRetry OnMethodError) Policy{
	policy.onMethodError = onRetry
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
					panicIfExceedLimit(policy, i+1, panicErr)
				}
			}()
			return funcBody()
		}
		funcReturn = recoverableMethod()
		if funcReturn.Err == nil && funcReturn.Valid && !panicOccurred {
			return
		}
		if policy.onFuncError != nil && !panicOccurred {
			policy.onFuncError(i, funcReturn.ReturnValue, funcReturn.Err)
		}
	}
	return
}

func(policy *policy) ExecuteMethod(methodBody Method) error {
	function := func() FuncReturn{
		var err = methodBody()
		return FuncReturn{nil, true, err}
	}
	var wrappedPolicy Policy = policy
	if policy.onMethodError != nil {
		wrappedPolicy = policy.SetOnFuncRetry(func(retryCount int, _ interface{}, err error){
			policy.onMethodError(retryCount, err)
		})
	}

	var funcReturn = wrappedPolicy.ExecuteFunc(function)
	return funcReturn.Err
}

func panicIfExceedLimit(policy *policy, i int, err interface{}) {
	if !(policy.retryOnPanic && policy.continueRetry(i)) {
		panic(err)
	}
}