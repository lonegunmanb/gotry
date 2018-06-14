package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"errors"
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
	onRetryHook := func(interface{}, error) {
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
	onRetryHook := func(interface{}, error) {
		suite.retried = true
	}
	var returnValue, isReturnValueValid, err = suite.policy.ExecuteWithRetryHook(errorFunc, onRetryHook)
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue)
	assert.True(suite.T(), isReturnValueValid)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.True(suite.T(), suite.retried)
}

func (suite *TryTestSuite) TestMultipleRetry() {
	const RetryAttempt = 5
	suite.policy.SetRetry(RetryAttempt)
	var retryCount = 0
	var errorFunc = func() (interface{}, bool, error) {
		return ExpectedReturnValue, true, ExpectedError
	}
	onRetryHook := func(interface{}, error) {
		retryCount++
	}
	var _, _, err = suite.policy.ExecuteWithRetryHook(errorFunc, onRetryHook)
	assert.Equal(suite.T(), ExpectedError, err)
	assert.Equal(suite.T(), RetryAttempt, retryCount)
}

func TestTrySuite(t *testing.T) {
	suite.Run(t, &TryTestSuite{})
}



