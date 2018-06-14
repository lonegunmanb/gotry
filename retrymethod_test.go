package gotry

import (
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"testing"
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
	var onRetryHook OnRetryMethod = func(int, error) {
		suite.retried = true
	}
	var err = suite.policy.ExecuteMethodWithRetryHook(errorMethod, onRetryHook)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.True(suite.T(), suite.retried)
}

func (suite *RetryMethodTestSuite) TestMultipleRetryMethod() {
	const RetryAttempt = 2
	suite.policy.SetRetry(RetryAttempt)
	var errorMethod = func() error {
		return ExpectedError
	}
	mockRetry, onRetryHook := prepareMockRetryMethodHook()
	var err = suite.policy.ExecuteMethodWithRetryHook(errorMethod, onRetryHook)
	assert.Equal(suite.T(), ExpectedError, err)
	assertTwiceRetryMethodCall(mockRetry, suite)
}

func prepareMockRetryMethodHook() (*mockRetry, func(retryAttempt int, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, err error) {
		mockRetry.OnRetryMethod(retryAttempt, err)
	}
	expectTwiceRetryMethodCall(mockRetry)
	return mockRetry, onRetryHook
}

func assertTwiceRetryMethodCall(mockRetry *mockRetry, suite *RetryMethodTestSuite) {
	mockRetry.AssertCalled(suite.T(), "OnRetryMethod", 1, ExpectedError)
	mockRetry.AssertCalled(suite.T(), "OnRetryMethod", 2, ExpectedError)
}

func expectTwiceRetryMethodCall(mockRetry *mockRetry) {
	mockRetry.On("OnRetryMethod", 1, ExpectedError).Return()
	mockRetry.On("OnRetryMethod", 2, ExpectedError).Return()
}