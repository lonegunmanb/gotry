package gotry

import (
	"github.com/stretchr/testify/suite"
	"errors"
	"github.com/stretchr/testify/mock"
)

const ExpectedReturnValue = 1

var ExpectedError = errors.New("expectedError")

type TryTestBaseSuite struct {
	suite.Suite
	policy *Policy
	retried bool
}

func (suite *TryTestBaseSuite) SetupTest() {
	suite.policy = &Policy{}
	suite.policy.SetRetry(1)
	suite.retried = false
}

type mockRetry struct {
	mock.Mock
}

func (hook *mockRetry) OnRetryFunc(retryAttempt int, returnValue interface{}, err error){
	hook.Called(retryAttempt, returnValue, err)
}

func (hook *mockRetry) OnRetryMethod(retryAttempt int, err error) {
	hook.Called(retryAttempt, err)
}
