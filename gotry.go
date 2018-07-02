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
	WithRetryLimit(retryLimit int) Policy
	WithInfiniteRetry() Policy
	WithRetryPredicate(predicate func()bool) Policy
	WithRetryOnPanic(retryOnPanic bool) Policy
	WithOnFuncRetry(onRetry OnFuncError) Policy
	WithOnMethodRetry(onRetry OnMethodError) Policy
	WithOnPanic(onPanic OnPanic) Policy
	ExecuteFunc(funcBody Func) FuncReturn
	ExecuteMethod(methodBody Method) error
}

type policy struct{
	retryOnPanic  bool
	shouldRetry   func(int) bool
	onFuncError   OnFuncError
	onMethodError OnMethodError
	onPanic       OnPanic
}

func NewPolicy() Policy {
	return &policy{
		retryOnPanic: true,
		shouldRetry:  func(retriedCount int) bool { return false },
	}
}

func (policy policy) WithRetryLimit(retryLimit int) Policy{
	policy.shouldRetry = func(retriedCount int) bool {
		return retriedCount <= retryLimit
	}
	return &policy
}

func (policy policy) WithInfiniteRetry() Policy {
	policy.shouldRetry = func(retriedCount int) bool {
		return true
	}
	return &policy
}

func (policy policy) WithRetryPredicate(predicate func()bool) Policy{
	policy.shouldRetry = func(int)bool {
		return predicate()
	}
	return &policy
}

func (policy policy) WithRetryOnPanic(retryOnPanic bool) Policy{
	policy.retryOnPanic = retryOnPanic
	return &policy
}

func (policy policy) WithOnFuncRetry(onRetry OnFuncError) Policy{
	policy.onFuncError = onRetry
	return &policy
}

func (policy policy) WithOnMethodRetry(onRetry OnMethodError) Policy{
	policy.onMethodError = onRetry
	return &policy
}

func (policy policy) WithOnPanic(onPanic OnPanic) Policy{
	policy.onPanic = onPanic
	return &policy
}

func(policy *policy) ExecuteFunc(funcBody Func) (funcReturn FuncReturn) {
	notifyPanic := policy.buildNotifyPanicMethod()
	for retried := 0; policy.shouldRetry(retried); retried++ {
		var recoverableBody = policy.wrapFuncBodyWithPanicNotify(notifyPanic, funcBody, retried)
		var panicOccurred bool
		funcReturn, panicOccurred = recoverableBody()
		if panicOccurred {
			continue
		}
		if success(funcReturn) {
			return
		}
		policy.onError(retried, funcReturn)
	}
	return
}

func (policy *policy) wrapFuncBodyWithPanicNotify(notifyPanic OnPanic, funcBody Func, i int)(func() (FuncReturn, bool)) {
	return func() (funcReturn FuncReturn, panicOccurred bool) {
		panicOccurred = false
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				panicOccurred = true
				notifyPanic(panicErr)
				panicIfExceedLimit(policy,
					nextIterationBecauseDeferExecuteAtLastSoIShouldIncreaseToJudgeIfPanicNeeded(i),
					panicErr)
			}
		}()
		funcReturn = funcBody()
		return
	}
}

func (policy *policy) buildNotifyPanicMethod() OnPanic {
	return func(panicError interface{}) {
		if policy.onPanic != nil {
			policy.onPanic(panicError)
		}
	}
}

func (policy *policy) onError(retryAttempted int, funcReturn FuncReturn) {
	if policy.onFuncError != nil{
		policy.onFuncError(retryAttempted, funcReturn.ReturnValue, funcReturn.Err)
	}
}

func success(funcReturn FuncReturn) bool {
	return funcReturn.Err == nil && funcReturn.Valid
}

func nextIterationBecauseDeferExecuteAtLastSoIShouldIncreaseToJudgeIfPanicNeeded(i int) int {
	return i + 1
}

func(policy *policy) ExecuteMethod(methodBody Method) error {
	function := methodBody.convertToFunc()
	var wrappedPolicy Policy = policy
	if policy.onMethodError != nil {
		wrappedPolicy = policy.wireOnFuncErrorToOnMethodError()
	}

	var funcReturn = wrappedPolicy.ExecuteFunc(function)
	return funcReturn.Err
}

func (policy *policy) wireOnFuncErrorToOnMethodError() Policy {
	return policy.WithOnFuncRetry(func(retryCount int, _ interface{}, err error) {
		policy.onMethodError(retryCount, err)
	})
}

func (methodBody Method) convertToFunc() (func() FuncReturn){
	return func() FuncReturn{
		var err = methodBody()
		return FuncReturn{nil, true, err}
	}
}

func panicIfExceedLimit(policy *policy, i int, err interface{}) {
	if !(policy.retryOnPanic && policy.shouldRetry(i)) {
		panic(err)
	}
}