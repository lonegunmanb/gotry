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
	mockRetry := &mockRetry{}
	onError := func(retryAttempt int, err error) {
		mockRetry.OnMethodError(retryAttempt, err)
	}
	err := suite.policy.WithOnMethodRetry(onError).ExecuteMethod(successMethod)
	assert.Nil(suite.T(), err)
	mockRetry.AssertNotCalled(suite.T(), OnMethodErrorMethodName)
}

func (suite *RetryMethodTestSuite) TestErrorMethod() {
	var errorMethod Method = func() error {
		return ExpectedError
	}
	var err = suite.policy.ExecuteMethod(errorMethod)
	assert.Equal(suite.T(), ExpectedError, err)
}

func (suite *RetryMethodTestSuite) TestMultipleRetryMethod() {
	const RetryAttempt = 2
	suite.policy = suite.policy.WithRetryLimit(RetryAttempt)
	var errorMethod = func() error {
		return ExpectedError
	}
	mockRetry, onError := prepareMockExpectingMethodRetry(RetryAttempt)
	var err = suite.policy.WithOnMethodRetry(onError).ExecuteMethod(errorMethod)
	assert.Equal(suite.T(), ExpectedError, err)
	assertCallOnMethodError(mockRetry, suite, RetryAttempt)
}

func (suite *RetryMethodTestSuite) TestInfiniteRetryMethod() {
	suite.policy = suite.policy.WithInfiniteRetry()
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
	mockRetry, onRetryHook := prepareMockExpectingMethodRetry(2)
	var err = suite.policy.WithOnMethodRetry(onRetryHook).ExecuteMethod(errorMethod)
	assert.Nil(suite.T(), err)
	assertCallOnMethodError(mockRetry, suite, 2)
}

func (suite *RetryMethodTestSuite) TestOnPanicMethodWithoutOnError() {
	mockRetry, onPanic := prepareMockOnPanicMethodWithoutOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertCalled(suite.T(), OnPanicMethodName, PanicContent)
	}()
	suite.policy.WithOnPanic(onPanic).ExecuteMethod(panicMethod)
}

func (suite *RetryMethodTestSuite) TestOnPanicMethodWithOnError() {
	mockRetry, onError, onPanic := prepareMockOnPanicMethodWithOnError()
	defer func(){
		_ = recover()
		mockRetry.AssertNotCalled(suite.T(), OnMethodErrorMethodName, mock.Anything, mock.Anything)
	}()
	suite.policy.WithOnMethodRetry(onError).WithOnPanic(onPanic).ExecuteMethod(panicMethod)
}

func prepareMockOnPanicMethodWithoutOnError() (*mockRetry, func(interface{})){
	mockRetry := &mockRetry{}
	onPanicHook := func(panicError interface{}){
		mockRetry.OnPanic(panicError)
	}
	mockRetry.On(OnPanicMethodName, PanicContent).Return()
	return mockRetry, onPanicHook
}

func prepareMockOnPanicMethodWithOnError() (*mockRetry, OnMethodError, OnPanic){
	mockRetry, onPanic := prepareMockOnPanicFuncWithoutOnError()
	mockRetry.On(OnMethodErrorMethodName, mock.Anything, mock.Anything).Return()
	onErrorRetry := func(retryCount int, err error){
		mockRetry.OnMethodError(0, nil)
	}
	return mockRetry, onErrorRetry, onPanic
}

func prepareMockExpectingMethodRetry(expectingRetryCount int) (*mockRetry, func(retryAttempt int, err error)) {
	return prepareMockRetry(expectingRetryCount+1)
}

func prepareMockRetry(expectCount int) (*mockRetry, func(retryAttempt int, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, err error) {
		mockRetry.OnMethodError(retryAttempt, err)
	}
	setupOnMethodError(mockRetry, expectCount)
	return mockRetry, onRetryHook
}

func assertCallOnMethodError(mockRetry *mockRetry, suite *RetryMethodTestSuite, expectingCallCount int){
	for i:=0; i < expectingCallCount; i++ {
		mockRetry.AssertCalled(suite.T(), OnMethodErrorMethodName, i, ExpectedError)
	}
}

func setupOnMethodError(mockRetry *mockRetry, expectCount int) {
	for i:=0; i < expectCount; i++ {
		mockRetry.On(OnMethodErrorMethodName, i, ExpectedError).Return()
	}
}