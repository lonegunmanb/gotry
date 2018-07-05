package gotry

import (
	"time"
	"errors"
)

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
	WithRetryPredicate(predicate func(int)bool) Policy
	WithRetryOnPanic(retryOnPanic bool) Policy
	WithOnFuncRetry(onRetry OnFuncError) Policy
	WithOnMethodRetry(onRetry OnMethodError) Policy
	WithOnPanic(onPanic OnPanic) Policy
	TryFunc(funcBody Func) FuncReturn
	TryMethod(methodBody Method) error
	TryFuncWithCancellation(funcBody func() FuncReturn, cancellation Cancellation) FuncReturn
	TryMethodWithCancellation(methodBody Method, cancellation Cancellation) error
	TryFuncWithTimeout(funcBody func() FuncReturn, duration time.Duration) FuncReturn
	TryMethodWithTimeout(methodBody Method, duration time.Duration) error
}

type policy struct{
	retryOnPanic  bool
	shouldRetry   func(int) bool
	onFuncError   OnFuncError
	onMethodError OnMethodError
	onPanic       OnPanic
}

var TimeoutError = errors.New("timeout")

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

func (policy policy) WithRetryPredicate(predicate func(int)bool) Policy{
	policy.shouldRetry = func(retriedCount int)bool {
		return predicate(retriedCount)
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

func (policy *policy) TryFuncWithTimeout(funcBody func() FuncReturn, duration time.Duration) FuncReturn{
	timeoutCancellation := &cancellation{}
	funcReturnChan := make(chan FuncReturn)
	go func(){
		funcReturnChan <- policy.TryFuncWithCancellation(funcBody, timeoutCancellation)
	}()
	select {
		case funcReturn := <- funcReturnChan: return funcReturn
		case <- time.After(duration):{
			timeoutCancellation.Cancel()
			return FuncReturn{Valid:false, Err:TimeoutError}
		}
	}
}

func (policy *policy) TryFuncWithCancellation(funcBody func() FuncReturn, cancellation Cancellation) FuncReturn{
	return policy.withCancellation(cancellation).TryFunc(funcBody)
}

func(policy *policy) TryFunc(funcBody Func) (funcReturn FuncReturn) {
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

func (policy *policy) TryMethodWithTimeout(methodBody Method, duration time.Duration) error{
	timeoutCancellation := &cancellation{}
	errChan := make(chan error)
	go func(){
		errChan <- policy.TryMethodWithCancellation(methodBody, timeoutCancellation)
	}()
	select {
		case err := <-errChan: return err
		case <- time.After(duration):{
			timeoutCancellation.Cancel()
			return TimeoutError
		}
	}
}

func (policy *policy) TryMethodWithCancellation(methodBody Method, cancellation Cancellation) error{
	return policy.withCancellation(cancellation).TryMethod(methodBody)
}

func(policy *policy) TryMethod(methodBody Method) error {
	function := methodBody.convertToFunc()
	var wrappedPolicy Policy = policy
	if policy.onMethodError != nil {
		wrappedPolicy = policy.wireOnFuncErrorToOnMethodError()
	}

	var funcReturn = wrappedPolicy.TryFunc(function)
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

func (policy policy) withCancellation(cancellation Cancellation) Policy{
	originPredicate := policy.shouldRetry
	return policy.WithRetryPredicate(func(retriedCount int) bool {
		return !cancellation.IsCancellationRequested() && originPredicate(retriedCount)
	})
}