package gotry

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

var successFunc Func = func() FuncReturn {
	return FuncReturn{ExpectedReturnValue, true, nil}
}

var errorFunc = func() FuncReturn {
	return FuncReturn{ExpectedReturnValue, true, ExpectedError}
}

type RetryFuncTestSuite struct {
	TryTestBaseSuite
}

func TestRetryFuncSuite(t *testing.T) {
	suite.Run(t, &RetryFuncTestSuite{})
}

func (suite *RetryFuncTestSuite) TestSuccessFunc(){
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncError(retryAttempt, returnValue, err)
	}
	funcReturn := suite.policy.WithOnFuncRetry(onError).TryFunc(successFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	mockRetry.AssertNotCalled(suite.T(), OnFuncErrorMethodName)
}

func (suite *RetryFuncTestSuite) TestInvalidReturnValueFunc() {
	var invalidReturnFunc Func = func() FuncReturn {
		var returnValue = ExpectedReturnValue
		return FuncReturn{returnValue, false, nil}
	}
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncError(retryAttempt, returnValue, err)
	}
	mockRetry.On(OnFuncErrorMethodName, 0, ExpectedReturnValue, nil).Return()
	mockRetry.On(OnFuncErrorMethodName, 1, ExpectedReturnValue, nil).Return()
	funcReturn := suite.policy.WithOnFuncRetry(onError).TryFunc(invalidReturnFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assert.False(suite.T(), funcReturn.Valid, "invalid return value should cause retry")
	assert.Equal(suite.T(), ExpectedReturnValue, funcReturn.ReturnValue,
		"invalid return value should be returned")
	mockRetry.AssertCalled(suite.T(), OnFuncErrorMethodName, 0, ExpectedReturnValue, nil)
	mockRetry.AssertCalled(suite.T(), OnFuncErrorMethodName, 1, ExpectedReturnValue, nil)
	mockRetry.AssertNumberOfCalls(suite.T(), OnFuncErrorMethodName, 2)
}

func (suite *RetryFuncTestSuite) TestErrorFunc() {
	var funcReturn = suite.policy.TryFunc(errorFunc)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
}

func assertValidReturnValue(suite *RetryFuncTestSuite, returnValue interface{}, isReturnValueValid bool) {
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
}

func (suite *RetryFuncTestSuite) TestMultipleRetry() {
	const RetryAttempt = 2
	suite.policy = suite.policy.WithRetryLimit(RetryAttempt)
	var errorFunc = func() FuncReturn {
		return FuncReturn{ExpectedReturnValue, true, ExpectedError}
	}
	mockRetry, onRetryHook := prepareMockWithOnError(2)
	suite.policy = suite.policy.WithOnFuncRetry(onRetryHook)
	var funcReturn = suite.policy.TryFunc(errorFunc)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
	assertCallOnFuncError(mockRetry, suite, RetryAttempt)
}

func (suite *RetryFuncTestSuite) TestInfiniteRetry() {
	suite.policy = suite.policy.WithInfiniteRetry()
	invokeCount := 0
	var errorFunc = func() FuncReturn {
		defer func(){
			invokeCount++
		}()
		if invokeCount < 2 {
			return FuncReturn{ExpectedReturnValue, true, ExpectedError}
		}
		return FuncReturn{ExpectedReturnValue, true, nil}
	}
	mockRetry, onRetryHook := prepareMockWithOnError(2)
	suite.policy = suite.policy.WithOnFuncRetry(onRetryHook)
	var funcReturn = suite.policy.TryFunc(errorFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assertCallOnFuncError(mockRetry, suite, 2)
}

func (suite *RetryFuncTestSuite) TestPanicFunc() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.WithOnPanic(onPanic)
	defer func (){
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		//First run invoke once, then retry invoke twice
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicEventEvenNoRetryOnPanicFunc() {
	var emptyOnRetry= func(retryCount int, returnValue interface{}, err error) {}
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.WithRetryOnPanic(false).WithOnFuncRetry(emptyOnRetry).WithOnPanic(onPanic)
	defer func() {
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 1)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithNoRetryEvent(){
	defer func(){
		panicError := recover()
		assert.Equal(suite.T(), PanicContent, panicError)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestNotRetryOnPanicFunc() {
	suite.policy = suite.policy.WithRetryOnPanic(false)
	defer func(){
		err := recover()
		assert.Equal(suite.T(), PanicContent, err)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.WithOnPanic(onPanic)
	defer func(){
		_ = recover()
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicShouldNotCallOnErrorEvent() {
	mockRetry, onError, onPanic := prepareMockOnPanicFuncWithOnError()
	suite.policy = suite.policy.WithOnFuncRetry(onError).WithOnPanic(onPanic)
	defer func(){
		_ = recover()
		mockRetry.AssertNotCalled(suite.T(), OnFuncErrorMethodName, mock.Anything, mock.Anything, mock.Anything)
	}()
	suite.policy.TryFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnRetryPredicate() {
	retried := false
	predicate := func(int)bool{
		defer func(){
			retried = true
		}()
		return !retried
	}
	var invalidReturnFunc Func = func() FuncReturn {
		var returnValue = ExpectedReturnValue
		return FuncReturn{returnValue, false, nil}
	}
	suite.policy = suite.policy.WithRetryPredicate(predicate)
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncError(retryAttempt, returnValue, err)
	}
	mockRetry.On(OnFuncErrorMethodName, 0, ExpectedReturnValue, nil).Return()
	_ = suite.policy.WithOnFuncRetry(onError).TryFunc(invalidReturnFunc)
	assert.True(suite.T(), retried)
	mockRetry.AssertNumberOfCalls(suite.T(), OnFuncErrorMethodName, 1)
}

func (suite *RetryFuncTestSuite) TestCancelFuncRetry(){
	var cancellation Cancellation = &cancellation{}
	retried := false
	suite.policy = suite.policy.WithInfiniteRetry().WithOnFuncRetry(
		func(retriedCount int, returnValue interface{}, err error){
			retried = true
			cancellation.Cancel()
	})
	errChan := make(chan FuncReturn)
	defer close(errChan)
	go func(){
		errChan <- suite.policy.TryFuncWithCancellation(errorFunc, cancellation)
	}()
	select {
		case funcReturn := <- errChan: {
			assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
			assert.True(suite.T(), retried)
			assert.True(suite.T(), cancellation.IsCancellationRequested())
		}
		case <- time.After(time.Millisecond * 50): assert.Fail(suite.T(), "timeout")
	}
}

func (suite *RetryFuncTestSuite) TestInfiniteRetryFuncWithTimeout(){
	retried := false
	suite.policy = suite.policy.WithInfiniteRetry().WithOnFuncRetry(
		func(retriedCount int, returnValue interface{}, err error){
			retried = true
	})
	errChan := make(chan FuncReturn)
	go func(){
		errChan <- suite.policy.TryFuncWithTimeout(errorFunc, time.Millisecond * 10)
	}()
	select {
		case funcReturn := <- errChan: {
			assert.Equal(suite.T(), TimeoutError, funcReturn.Err)
			assert.False(suite.T(), funcReturn.Valid)
			assert.True(suite.T(), retried)
		}
		case <- time.After(time.Millisecond * 50): assert.Fail(suite.T(), "timeout")
	}
}

func (suite *RetryFuncTestSuite) TestRetryFuncWithTimeout(){
	suite.policy = suite.policy.WithInfiniteRetry()
	returnChan := make(chan FuncReturn)
	go func(){
		returnChan <- suite.policy.TryFuncWithTimeout(successFunc, time.Millisecond * 10)
	}()
	select {
	case funcReturn := <- returnChan: {
			assert.Nil(suite.T(), funcReturn.Err)
			assert.True(suite.T(), funcReturn.Valid)
			assert.Equal(suite.T(), funcReturn.ReturnValue, ExpectedReturnValue)
		}
		case <- time.After(time.Millisecond * 50): assert.Fail(suite.T(), "timeout")
	}
}

func prepareMockOnPanicFuncWithoutOnError() (*mockRetry, func(interface{})){
	mockRetry := &mockRetry{}
	onPanicHook := func(panicError interface{}){
		mockRetry.OnPanic(panicError)
	}
	mockRetry.On(OnPanicMethodName, PanicContent).Return()
	return mockRetry, onPanicHook
}

func prepareMockOnPanicFuncWithOnError() (*mockRetry, OnFuncError, OnPanic){
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	mockRetry.On(OnFuncErrorMethodName, mock.Anything, mock.Anything, mock.Anything).Return()
	onErrorRetry := func(retryCount int, returnValue interface{}, err error){
		mockRetry.OnFuncError(0, nil, nil)
	}
	return mockRetry, onErrorRetry, onPanic
}

func prepareMockWithOnError(expectingRetryCount int)(*mockRetry, func(retryAttempt int, returnValue interface{}, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncError(retryAttempt, returnValue, err)
	}
	setupFuncRetry(mockRetry, expectingRetryCount)
	return mockRetry, onRetryHook
}

func assertCallOnFuncError(mock *mockRetry, suite *RetryFuncTestSuite, expectingCallCount int){
	for i:=0; i < expectingCallCount; i++ {
		mock.AssertCalled(suite.T(), OnFuncErrorMethodName, i, ExpectedReturnValue, ExpectedError)
	}
}

func setupFuncRetry(mockRetry *mockRetry, expectingRetryCount int) {
	for i:=0; i < expectingRetryCount+1; i++{
		mockRetry.On(OnFuncErrorMethodName, i, ExpectedReturnValue, ExpectedError).Return()
	}
}