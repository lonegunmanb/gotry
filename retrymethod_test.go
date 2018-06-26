package gotry

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/stretchr/testify/mock"
)

type RetryMethodTestSuite struct {
	TryTestBaseSuite
}

func TestRetryMethodSuite(t *testing.T) {
	suite.Run(t, &RetryMethodTestSuite{})
}

func (suite *RetryMethodTestSuite) TestSuccessMethod(){
	var successMethod Method = func() error {
		return nil
	}
	err := suite.policy.ExecuteMethod(successMethod)
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), suite.retried)
}

func (suite *RetryMethodTestSuite) TestErrorMethod() {
	var errorMethod Method = func() error {
		return ExpectedError
	}
	var onRetryHook OnMethodErrorRetry = func(int, error) {
		suite.retried = true
	}
	var err = suite.policy.ExecuteMethodWithRetryHook(errorMethod, onRetryHook, nil)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.True(suite.T(), suite.retried)
}

func (suite *RetryMethodTestSuite) TestMultipleRetryMethod() {
	const RetryAttempt = 2
	suite.policy = suite.policy.SetRetry(RetryAttempt)
	var errorMethod = func() error {
		return ExpectedError
	}
	mockRetry, onRetryHook := prepareMockRetryMethodHook()
	var err = suite.policy.ExecuteMethodWithRetryHook(errorMethod, onRetryHook, nil)
	assert.Equal(suite.T(), ExpectedError, err)
	assertTwiceRetryMethodCall(mockRetry, suite)
}

func (suite *RetryMethodTestSuite) TestInfiniteRetryMethod() {
	suite.policy = suite.policy.SetInfiniteRetry(true)
	count := 0
	var errorMethod = func() error {
		defer func(){
			count++
		}()
		if count< 2 {
			return ExpectedError
		}
		return nil
	}
	mockRetry, onRetryHook := prepareMockRetryMethodHook()
	var err = suite.policy.ExecuteMethodWithRetryHook(errorMethod, onRetryHook, nil)
	assert.Nil(suite.T(), err)
	assertTwiceRetryMethodCall(mockRetry, suite)
}

func (suite *RetryMethodTestSuite) TestOnPanicMethodWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicMethodWithoutOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
	}()
	suite.policy.ExecuteMethodWithRetryHook(panicMethod, nil, onPanic)
}

func (suite *RetryMethodTestSuite) TestOnPanicMethodWithOnError() {
	mockRetry, onError, onPanic := prepareMockOnPanicMethodWithOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertNotCalled(suite.T(), OnMethodErrorRetryMethodName, mock.Anything, mock.Anything)
	}()
	suite.policy.ExecuteMethodWithRetryHook(panicMethod, onError, onPanic)
}

func prepareMockOnPanicMethodWithoutOnError() (*mockRetry, func(interface{})){
	mockRetry := &mockRetry{}
	onPanicHook := func(panicError interface{}){
		mockRetry.OnPanic(panicError)
	}
	mockRetry.On(OnPanicMethodName, PanicContent).Return()
	return mockRetry, onPanicHook
}

func prepareMockOnPanicMethodWithOnError() (*mockRetry, OnMethodErrorRetry, OnPanic){
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	mockRetry.On(OnMethodErrorRetryMethodName, mock.Anything, mock.Anything).Return()
	onErrorRetry := func(retryCount int, err error){
		mockRetry.OnMethodErrorRetry(0, nil)
	}
	return mockRetry, onErrorRetry, onPanic
}

func prepareMockRetryMethodHook() (*mockRetry, func(retryAttempt int, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, err error) {
		mockRetry.OnMethodErrorRetry(retryAttempt, err)
	}
	expectTwiceRetryMethodCall(mockRetry)
	return mockRetry, onRetryHook
}

func assertTwiceRetryMethodCall(mockRetry *mockRetry, suite *RetryMethodTestSuite) {
	mockRetry.AssertCalled(suite.T(), OnMethodErrorRetryMethodName, 1, ExpectedError)
	mockRetry.AssertCalled(suite.T(), OnMethodErrorRetryMethodName, 2, ExpectedError)
}

func expectTwiceRetryMethodCall(mockRetry *mockRetry) {
	mockRetry.On(OnMethodErrorRetryMethodName, 1, ExpectedError).Return()
	mockRetry.On(OnMethodErrorRetryMethodName, 2, ExpectedError).Return()
}