package gotry

import (
	"github.com/stretchr/testify/suite"
	"errors"
	"github.com/stretchr/testify/mock"
)

const ExpectedReturnValue = 1
const OnPanicMethodName = "OnPanic"
const OnMethodErrorMethodName = "OnMethodError"
const OnFuncErrorMethodName = "OnFuncError"
var ExpectedError = errors.New("expectedError")
const PanicContent ="test panic"
var panicMethod = func() error {
	panic(PanicContent)
}
var panicFunc = func() FuncReturn {
	panic(PanicContent)
}

type TryTestBaseSuite struct {
	suite.Suite
	policy Policy
}

func (suite *TryTestBaseSuite) SetupTest() {
	suite.policy = NewPolicy().SetRetry(1)
}

type mockRetry struct {
	mock.Mock
}

func (hook *mockRetry) OnFuncError(retryAttempt int, returnValue interface{}, err error){
	hook.Called(retryAttempt, returnValue, err)
}

func (hook *mockRetry) OnMethodError(retryAttempt int, err error) {
	hook.Called(retryAttempt, err)
}

func (hook *mockRetry) OnPanic(panicError interface{}){
	hook.Called(panicError)
}