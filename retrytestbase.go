package gotry

import (
	"github.com/stretchr/testify/suite"
	"errors"
	"github.com/stretchr/testify/mock"
	"time"
)

const ExpectedReturnValue = 1
const (
	OnPanicMethodName       = "OnPanic"
	OnMethodErrorMethodName = "OnMethodError"
	OnFuncErrorMethodName   = "OnFuncError"
	OnTimeoutMethodName     = "OnTimeout"
)
var ExpectedError = errors.New("expectedError")
const PanicContent ="test panic"
var timeout = time.Millisecond * 10
var waitTime = timeout * 2
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
	suite.policy = NewPolicy().WithRetryLimit(1)
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
func (hook *mockRetry) OnTimeout(duration time.Duration) {
	hook.Called(duration)
}