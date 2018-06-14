package gotry

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/stretchr/testify/mock"
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
		return returnValue, returnValue > 0, nil
	}
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteFunc(successFunc)
	assert.Nil(suite.T(), err)
	assertValidReturnValue(suite, returnValue, isReturnValueValid)
	assert.False(suite.T(), suite.retried)
}

func (suite *RetryFuncTestSuite) TestInvalidReturnValueFunc() {
	var invalidReturnFunc Func = func() (interface{}, bool, error) {
		var returnValue= ExpectedReturnValue
		return returnValue, returnValue < 0, nil
	}
	onRetryHook := func(int, interface{}, error) {
		suite.retried = true
	}
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteFuncWithRetryHook(invalidReturnFunc,
																				   onRetryHook, nil)
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), isReturnValueValid, "invalid return value should cause retry")
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue,
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
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assertValidReturnValue(suite, returnValue, isReturnValueValid)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.True(suite.T(), suite.retried)
}

func assertValidReturnValue(suite *RetryFuncTestSuite, returnValue interface{}, isReturnValueValid bool) {
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
}

func (suite *RetryFuncTestSuite) TestMultipleRetry() {
	const RetryAttempt = 2
	suite.policy.SetRetry(RetryAttempt)
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	mockRetry, onRetryHook := prepareMockRetryFuncHook()
	var _, _, err = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assert.Equal(suite.T(), ExpectedError, err)
	assertTwiceRetryFuncCall(mockRetry, suite)
}

func (suite *RetryFuncTestSuite) TestInfiniteRetry() {
	suite.policy.SetInfiniteRetry(true)
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
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteFuncWithRetryHook(errorFunc, onRetryHook, nil)
	assert.Nil(suite.T(), err)
	assertValidReturnValue(suite, returnValue, isReturnValueValid)
	assertTwiceRetryFuncCall(mockRetry, suite)
}

func (suite *RetryMethodTestSuite) TestPanicFunc() {
	defer func(){
		unexpectedRuntimeError := recover()
		assert.Nil(suite.T(), unexpectedRuntimeError)
	}()
	suite.policy.ExecuteFunc(panicFunc)
}

func (suite *RetryMethodTestSuite) TestOnPanicFuncWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, nil, onPanic)
	mockRetry.AssertCalled(suite.T(), "OnPanic", PanicContent)
}

func (suite *RetryMethodTestSuite) TestOnPanicFuncWithOnError() {
	mockRetry, onError, onPanic := prepareMockOnPanicFuncWithOnError()
	suite.policy.ExecuteFuncWithRetryHook(panicFunc, onError, onPanic)
	mockRetry.AssertNotCalled(suite.T(), "OnFuncErrorRetry", mock.Anything, mock.Anything, mock.Anything)
}

func prepareMockOnPanicFuncWithoutOnError() (*mockRetry, func(interface{})){
	mockRetry := &mockRetry{}
	onPanicHook := func(panicError interface{}){
		mockRetry.OnPanic(panicError)
	}
	mockRetry.On("OnPanic", PanicContent).Return()
	return mockRetry, onPanicHook
}

func prepareMockOnPanicFuncWithOnError() (*mockRetry, OnFuncErrorRetry, OnPanic){
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	mockRetry.On("OnFuncErrorRetry", mock.Anything, mock.Anything, mock.Anything).Return()
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
	mockRetry.AssertCalled(suite.T(), "OnFuncErrorRetry", 1, ExpectedReturnValue, ExpectedError)
	mockRetry.AssertCalled(suite.T(), "OnFuncErrorRetry", 2, ExpectedReturnValue, ExpectedError)
}

func expectTwiceRetryFuncCall(mockRetry *mockRetry) {
	mockRetry.On("OnFuncErrorRetry", 1, ExpectedReturnValue, ExpectedError).Return()
	mockRetry.On("OnFuncErrorRetry", 2, ExpectedReturnValue, ExpectedError).Return()
}