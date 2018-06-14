package gotry

import (
	"github.com/stretchr/testify/suite"
	"errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
	"testing"
)

const ExpectedReturnValue = 1

var ExpectedError = errors.New("expectedError")

type TryTestSuite struct {
	suite.Suite
	policy *Policy
	retried bool
}

func (suite *TryTestSuite) SetupTest() {
	suite.policy = &Policy{}
	suite.policy.SetRetry(1)
	suite.retried = false
}

func TestTrySuite(t *testing.T) {
	suite.Run(t, &TryTestSuite{})
}

func (suite *TryTestSuite) TestSuccessFunc(){
	var successFunc Func = func() (interface{}, bool, error) {
		var returnValue= ExpectedReturnValue
		return returnValue, returnValue > 0, nil
	}
	var returnValue, isReturnValueValid, err = suite.policy.Execute(successFunc)
	assert.Nil(suite.T(), err)
	assert.True(suite.T(), isReturnValueValid, "success invoke's return value should pass check")
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue, "should be equal")
	assert.False(suite.T(), suite.retried)
}

func (suite *TryTestSuite) TestInvalidReturnValueFunc() {
	var invalidReturnFunc Func = func() (interface{}, bool, error) {
		var returnValue= ExpectedReturnValue
		return returnValue, returnValue < 0, nil
	}
	onRetryHook := func(int, interface{}, error) {
		suite.retried = true
	}
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteWithRetryHook(invalidReturnFunc,
																				   onRetryHook)
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), isReturnValueValid, "invalid return value should cause retry")
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue,
		"invalid return value should be returned")
	assert.True(suite.T(), suite.retried)
}

func (suite *TryTestSuite) TestErrorFunc() {
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	onRetryHook := func(int, interface{}, error) {
		suite.retried = true
	}
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteWithRetryHook(errorFunc, onRetryHook)
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.True(suite.T(), suite.retried)
}

type mockRetry struct {
	mock.Mock
}

func (hook *mockRetry) OnRetry(retryAttempt int, returnValue interface{}, err error){
	hook.Called(retryAttempt, returnValue, err)
}

func (suite *TryTestSuite) TestMultipleRetry() {
	const RetryAttempt = 2
	suite.policy.SetRetry(RetryAttempt)
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	mockRetry, onRetryHook := prepareMockRetryHook()
	var _, _, err = suite.policy.ExecuteWithRetryHook(errorFunc, onRetryHook)
	assert.Equal(suite.T(), ExpectedError, err)
	assertTwiceRetryCall(mockRetry, suite)
}

func prepareMockRetryHook() (*mockRetry, func(retryAttempt int, returnValue interface{}, err error)) {
	mockRetry := &mockRetry{}
	onRetryHook := func(retryAttempt int, returnValue interface{}, err error) {
		mockRetry.OnRetry(retryAttempt, returnValue, err)
	}
	expectTwiceRetryCall(mockRetry)
	return mockRetry, onRetryHook
}

func assertTwiceRetryCall(mockRetry *mockRetry, suite *TryTestSuite) {
	mockRetry.AssertCalled(suite.T(), "OnRetry", 1, ExpectedReturnValue, ExpectedError)
	mockRetry.AssertCalled(suite.T(), "OnRetry", 2, ExpectedReturnValue, ExpectedError)
}

func expectTwiceRetryCall(mockRetry *mockRetry) {
	mockRetry.On("OnRetry", 1, ExpectedReturnValue, ExpectedError).Return()
	mockRetry.On("OnRetry", 2, ExpectedReturnValue, ExpectedError).Return()
}