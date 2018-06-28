package gotry

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type RetryFuncTestSuite struct {
	TryTestBaseSuite
}

func TestRetryFuncSuite(t *testing.T) {
	suite.Run(t, &RetryFuncTestSuite{})
}

func (suite *RetryFuncTestSuite) TestSuccessFunc(){
	var successFunc Func = func() FuncReturn {
		var returnValue= ExpectedReturnValue
		return FuncReturn{returnValue, true, nil}
	}
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncErrorRetry(retryAttempt, returnValue, err)
	}
	funcReturn := suite.policy.SetOnFuncRetry(onError).ExecuteFunc(successFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	mockRetry.AssertNotCalled(suite.T(), OnFuncErrorRetryMethodName)
}

func (suite *RetryFuncTestSuite) TestInvalidReturnValueFunc() {
	var invalidReturnFunc Func = func() FuncReturn {
		var returnValue = ExpectedReturnValue
		return FuncReturn{returnValue, false, nil}
	}
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncErrorRetry(retryAttempt, returnValue, err)
	}
	mockRetry.On(OnFuncErrorRetryMethodName, 1, ExpectedReturnValue, nil).Return()
	funcReturn := suite.policy.SetOnFuncRetry(onError).ExecuteFunc(invalidReturnFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assert.False(suite.T(), funcReturn.Valid, "invalid return value should cause retry")
	assert.Equal(suite.T(), ExpectedReturnValue, funcReturn.ReturnValue,
		"invalid return value should be returned")
	mockRetry.AssertCalled(suite.T(), OnFuncErrorRetryMethodName, 1, ExpectedReturnValue, nil)
	mockRetry.AssertNumberOfCalls(suite.T(), OnFuncErrorRetryMethodName, 1)
}

func (suite *RetryFuncTestSuite) TestErrorFunc() {
	var errorFunc = func() FuncReturn {
		return FuncReturn{ExpectedReturnValue, true, ExpectedError}
	}
	var funcReturn = suite.policy.ExecuteFunc(errorFunc)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
}

func assertValidReturnValue(suite *RetryFuncTestSuite, returnValue interface{}, isReturnValueValid bool) {
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
}

func (suite *RetryFuncTestSuite) TestMultipleRetry() {
	const RetryAttempt = 2
	suite.policy = suite.policy.SetRetry(RetryAttempt)
	var errorFunc = func() FuncReturn {
		return FuncReturn{ExpectedReturnValue, true, ExpectedError}
	}
	mockRetry, onRetryHook := prepareMockWithOnError(2)
	suite.policy = suite.policy.SetOnFuncRetry(onRetryHook)
	var funcReturn = suite.policy.ExecuteFunc(errorFunc)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
	assertCallOnFuncError(mockRetry, suite, RetryAttempt)
}

func (suite *RetryFuncTestSuite) TestInfiniteRetry() {
	suite.policy = suite.policy.SetInfiniteRetry()
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
	suite.policy = suite.policy.SetOnFuncRetry(onRetryHook)
	var funcReturn = suite.policy.ExecuteFunc(errorFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assertCallOnFuncError(mockRetry, suite, 2)
}

func (suite *RetryFuncTestSuite) TestPanicFunc() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.SetOnPanic(onPanic)
	defer func (){
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		//First run invoke once, then retry invoke twice
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicEventEvenNoRetryOnPanicFunc() {
	var emptyOnRetry= func(retryCount int, returnValue interface{}, err error) {}
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.SetRetryOnPanic(false).SetOnFuncRetry(emptyOnRetry).SetOnPanic(onPanic)
	defer func() {
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 1)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithNoRetryEvent(){
	defer func(){
		panicError := recover()
		assert.Equal(suite.T(), PanicContent, panicError)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestNotRetryOnPanicFunc() {
	suite.policy = suite.policy.SetRetryOnPanic(false)
	defer func(){
		err := recover()
		assert.Equal(suite.T(), PanicContent, err)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy = suite.policy.SetOnPanic(onPanic)
	defer func(){
		_ = recover()
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicShouldNotCallOnErrorEvent() {
	mockRetry, onError, onPanic := prepareMockOnPanicFuncWithOnError()
	suite.policy = suite.policy.SetOnFuncRetry(onError).SetOnPanic(onPanic)
	defer func(){
		_ = recover()
		mockRetry.AssertNotCalled(suite.T(), OnFuncErrorRetryMethodName, mock.Anything, mock.Anything, mock.Anything)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func prepareMockOnPanicFuncWithoutOnError() (*mockRetry, func(interface{})){
	mockRetry := &mockRetry{}
	onPanicHook := func(panicError interface{}){
		mockRetry.OnPanic(panicError)
	}
	mockRetry.On(OnPanicMethodName, PanicContent).Return()
	return mockRetry, onPanicHook
}

func prepareMockOnPanicFuncWithOnError() (*mockRetry, OnFuncErrorRetry, OnPanic){
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	mockRetry.On(OnFuncErrorRetryMethodName, mock.Anything, mock.Anything, mock.Anything).Return()
	onErrorRetry := func(retryCount int, returnValue interface{}, err error){
		mockRetry.OnFuncErrorRetry(0, nil, nil)
	}
	return mockRetry, onErrorRetry, onPanic
}

func prepareMockWithOnError(expectingRetryCount int)(*mockRetry, func(retryAttempt int, returnValue interface{}, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncErrorRetry(retryAttempt, returnValue, err)
	}
	setupFuncRetry(mockRetry, expectingRetryCount)
	return mockRetry, onRetryHook
}

func assertCallOnFuncError(mock *mockRetry, suite *RetryFuncTestSuite, expectingCallCount int){
	for i:=0; i < expectingCallCount; i++ {
		mock.AssertCalled(suite.T(), OnFuncErrorRetryMethodName, i+1, ExpectedReturnValue, ExpectedError)
	}
}

func setupFuncRetry(mockRetry *mockRetry, expectingRetryCount int) {
	for i:=0; i < expectingRetryCount; i++{
		mockRetry.On(OnFuncErrorRetryMethodName, i+1, ExpectedReturnValue, ExpectedError).Return()
	}
}