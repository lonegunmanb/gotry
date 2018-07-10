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
type OnTimeout func(timeout time.Duration)

type FuncReturn struct {
	ReturnValue interface{}
	Valid       bool
	Err         error
}

type Policy interface {
	WithRetryLimit(retryLimit int) Policy
	WithRetryForever() Policy
	WithRetryUntil(stopPredicate func(int) bool) Policy
	WithLetItPanic() Policy
	WithTimeout(timeout time.Duration) Policy
	WithOnFuncRetry(onRetry OnFuncError) Policy
	WithOnMethodRetry(onRetry OnMethodError) Policy
	WithOnPanic(onPanic OnPanic) Policy
	TryFunc(funcBody Func) FuncReturn
	TryMethod(methodBody Method) error
	TryFuncWithCancellation(funcBody Func, cancellation Cancellation) FuncReturn
	TryMethodWithCancellation(methodBody Method, cancellation Cancellation) error
	WithOnTimeout(onTimeout OnTimeout) Policy
}

type policy struct{
	retryOnPanic  bool
	timeout       *time.Duration
	shouldRetry   func(int) bool
	funcExecutor  func(*policy, Func) FuncReturn
	onFuncError   OnFuncError
	onMethodError OnMethodError
	onPanic       OnPanic
	onTimeout     OnTimeout
}

var TimeoutError = errors.New("timeout")

func NewPolicy() Policy {
	policy := policy{
		retryOnPanic: true,
		shouldRetry:  func(retriedCount int) bool { return false },
	}
	policy.funcExecutor = directTryFunc
	return &policy
}

func (p policy) WithRetryLimit(retryLimit int) Policy{
	p.shouldRetry = func(retriedCount int) bool {
		return retriedCount <= retryLimit
	}
	return &p
}

func (p policy) WithRetryForever() Policy {
	p.shouldRetry = func(retriedCount int) bool {
		return true
	}
	return &p
}

func (p policy) WithRetryUntil(stopPredicate func(int)bool) Policy{
	p.shouldRetry = func(retriedCount int)bool {
		return !stopPredicate(retriedCount)
	}
	return &p
}

func (p policy) WithLetItPanic() Policy{
	p.retryOnPanic = false
	return &p
}

func (p policy) WithOnFuncRetry(onRetry OnFuncError) Policy{
	p.onFuncError = onRetry
	return &p
}

func (p policy) WithOnMethodRetry(onRetry OnMethodError) Policy{
	p.onMethodError = onRetry
	return &p
}

func (p policy) WithOnPanic(onPanic OnPanic) Policy{
	p.onPanic = onPanic
	return &p
}

func (p policy) WithTimeout(timeout time.Duration) Policy{
	p.timeout = &timeout
	p.funcExecutor = func(policy *policy, funcBody Func) FuncReturn {
		return policy.tryFuncWithTimeout(funcBody, timeout)
	}
	return &p
}

func (p policy) WithOnTimeout(onTimeout OnTimeout) Policy{
	p.onTimeout = onTimeout
	return &p
}

func (p *policy) tryFuncWithTimeout(funcBody Func, duration time.Duration) FuncReturn {
	timeoutCancellation := &cancellation{}
	funcReturnChan := make(chan FuncReturn)
	go func() {
		funcReturnChan <- p.TryFuncWithCancellation(funcBody, timeoutCancellation)
	}()
	select {
	case funcReturn := <-funcReturnChan:
		{
			return funcReturn
		}
	case <-time.After(duration):
		{
			timeoutCancellation.Cancel()
			notifyOnTimeout(p, duration)
			return FuncReturn{Valid: false, Err: TimeoutError}
		}
	}
}

func notifyOnTimeout(p *policy, duration time.Duration) {
	if p.onTimeout != nil {
		p.onTimeout(duration)
	}
}

func (p *policy) TryFuncWithCancellation(funcBody Func, cancellation Cancellation) FuncReturn{
	return p.tryFuncWithCancellation(funcBody, cancellation, directTryFunc)
}

func (p *policy) tryFuncWithCancellation(funcBody Func,
										 cancellation Cancellation,
										 tryExecutor func(*policy, Func) FuncReturn) FuncReturn {
	return tryExecutor(p.withCancellation(cancellation).(*policy), funcBody)
}

func(p *policy) TryFunc(funcBody Func) (funcReturn FuncReturn) {
	return p.funcExecutor(p, funcBody)
}

func directTryFunc(policy *policy, funcBody Func) (funcReturn FuncReturn) {
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

func (p *policy) wrapFuncBodyWithPanicNotify(notifyPanic OnPanic, funcBody Func, retried int)(func() (FuncReturn, bool)) {
	return func() (funcReturn FuncReturn, panicOccurred bool) {
		panicOccurred = false
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				panicOccurred = true
				notifyPanic(panicErr)
				panicIfExceedLimit(p,
					nextIterationBecauseDeferExecuteAtLastSoIShouldIncreaseToJudgeIfPanicNeeded(retried),
					panicErr)
			}
		}()
		funcReturn = funcBody()
		return
	}
}

func (p *policy) buildNotifyPanicMethod() OnPanic {
	return func(panicError interface{}) {
		if p.onPanic != nil {
			p.onPanic(panicError)
		}
	}
}

func (p *policy) onError(retryAttempted int, funcReturn FuncReturn) {
	if p.onFuncError != nil{
		p.onFuncError(retryAttempted, funcReturn.ReturnValue, funcReturn.Err)
	}
}

func success(funcReturn FuncReturn) bool {
	return funcReturn.Err == nil && funcReturn.Valid
}

func nextIterationBecauseDeferExecuteAtLastSoIShouldIncreaseToJudgeIfPanicNeeded(i int) int {
	return i + 1
}

func (p *policy) TryMethodWithCancellation(methodBody Method, cancellation Cancellation) error{
	return p.withCancellation(cancellation).TryMethod(methodBody)
}

func(p *policy) TryMethod(methodBody Method) error {
	function := methodBody.convertToFunc()
	var wrappedPolicy Policy = p
	if p.onMethodError != nil {
		wrappedPolicy = p.wireOnFuncErrorToOnMethodError()
	}

	var funcReturn = wrappedPolicy.TryFunc(function)
	return funcReturn.Err
}

func (p *policy) wireOnFuncErrorToOnMethodError() Policy {
	return p.WithOnFuncRetry(func(retryCount int, _ interface{}, err error) {
		p.onMethodError(retryCount, err)
	})
}

func (methodBody Method) convertToFunc() Func{
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

func (p policy) withCancellation(cancellation Cancellation) Policy{
	shouldRetry := p.shouldRetry
	return p.WithRetryUntil(func(retriedCount int) bool {
		return cancellation.IsCancellationRequested() || !shouldRetry(retriedCount)
	})
}