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
	var successFunc Func = func() (interface{}, bool, error) {
		var returnValue= ExpectedReturnValue
		return returnValue, true, nil
	}
	funcReturn := suite.policy.ExecuteFunc(successFunc)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assert.False(suite.T(), suite.retried)
}

func (suite *RetryFuncTestSuite) TestInvalidReturnValueFunc() {
	var invalidReturnFunc Func = func() (interface{}, bool, error) {
		var returnValue= ExpectedReturnValue
		return returnValue, false, nil
	}
	onRetryHook := func(int, interface{}, error) {
		suite.retried = true
	}
	funcReturn := suite.policy.ExecuteFuncWithRetryHook(invalidReturnFunc, onRetryHook, nil)
	assert.Nil(suite.T(), funcReturn.Err)
	assert.False(suite.T(), funcReturn.Valid, "invalid return value should cause retry")
	assert.Equal(suite.T(), ExpectedReturnValue, funcReturn.ReturnValue,
		"invalid return value should be returned")
	assert.True(suite.T(), suite.retried)
}

func (suite *RetryFuncTestSuite) TestErrorFunc() {
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	onRetryHook := func(int, interface{}, error) {
		suite.retried = true
	}
	var funcReturn = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
	assert.True(suite.T(), suite.retried)
}

func assertValidReturnValue(suite *RetryFuncTestSuite, returnValue interface{}, isReturnValueValid bool) {
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
}

func (suite *RetryFuncTestSuite) TestMultipleRetry() {
	const RetryAttempt = 2
	suite.policy = suite.policy.SetRetry(RetryAttempt)
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	mockRetry, onRetryHook := prepareMockRetryFuncHook()
	var funcReturn = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assert.Equal(suite.T(), ExpectedError, funcReturn.Err)
	assertTwiceRetryFuncCall(mockRetry, suite)
}

func (suite *RetryFuncTestSuite) TestInfiniteRetry() {
	suite.policy = suite.policy.SetInfiniteRetry(true)
	invokeCount := 0
	var errorFunc = func() (interface{}, bool, error) {
		defer func(){
			invokeCount++
		}()
		if invokeCount < 2 {
			return ExpectedReturnValue, true, ExpectedError
		}
		return ExpectedReturnValue, true, nil
	}
	mockRetry, onRetryHook := prepareMockRetryFuncHook()
	var funcReturn = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assert.Nil(suite.T(), funcReturn.Err)
	assertValidReturnValue(suite, funcReturn.ReturnValue, funcReturn.Valid)
	assertTwiceRetryFuncCall(mockRetry, suite)
}

func (suite *RetryFuncTestSuite) TestPanicFunc() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	defer func (){
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		//First run invoke once, then retry invoke twice
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, nil, onPanic)
}

func (suite *RetryFuncTestSuite) TestOnPanicEventEvenNoRetryOnPanicFunc() {
	suite.policy = suite.policy.SetRetryOnPanic(false)
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	defer func (){
		unexpectedRuntimeError := recover()
		assert.Equal(suite.T(), PanicContent, unexpectedRuntimeError)
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 1)
	}()
	var emptyOnRetry = func(retryCount int, returnValue interface{}, err error){}
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, emptyOnRetry, onPanic)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithNoRetryEvent(){
	defer func(){
		panicError := recover()
		assert.Equal(suite.T(), PanicContent, panicError)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestNotRetryOnPanicFunc() {
	suite.policy.SetRetryOnPanic(false)
	defer func(){
		err := recover()
		assert.Equal(suite.T(), PanicContent, err)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryFuncTestSuite) TestOnPanicFuncWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
		mockRetry.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
	}()
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, nil, onPanic)
}

func (suite *RetryFuncTestSuite) TestOnPanicShouldNotCallOnErrorEvent() {
	mockRetry, onError, onPanic := prepareMockOnPanicFuncWithOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertNotCalled(suite.T(), OnFuncErrorRetryMethodName, mock.Anything, mock.Anything, mock.Anything)
	}()
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, onError, onPanic)
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

func prepareMockRetryFuncHook() (*mockRetry, func(retryAttempt int, returnValue interface{}, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnFuncErrorRetry(retryAttempt, returnValue, err)
	}
	expectTwiceRetryFuncCall(mockRetry)
	return mockRetry, onRetryHook
}

func assertTwiceRetryFuncCall(mockRetry *mockRetry, suite *RetryFuncTestSuite) {
	mockRetry.AssertCalled(suite.T(), OnFuncErrorRetryMethodName, 1, ExpectedReturnValue, ExpectedError)
	mockRetry.AssertCalled(suite.T(), OnFuncErrorRetryMethodName, 2, ExpectedReturnValue, ExpectedError)
}

func expectTwiceRetryFuncCall(mockRetry *mockRetry) {
	mockRetry.On(OnFuncErrorRetryMethodName, 1, ExpectedReturnValue, ExpectedError).Return()
	mockRetry.On(OnFuncErrorRetryMethodName, 2, ExpectedReturnValue, ExpectedError).Return()
}