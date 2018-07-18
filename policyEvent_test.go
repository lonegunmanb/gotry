package gotry

import (
	"testing"
	"github.com/stretchr/testify/suite"
	"time"
)

type policyEventTestSuite struct {
	suite.Suite
	mock *mockRetry
	p *policy
}

func TestPolicyEvent(t *testing.T){
	suite.Run(t, &policyEventTestSuite{})
}

func (suite *policyEventTestSuite) SetupTest(){
	suite.p = &policy{}
	suite.mock = &mockRetry{}
	suite.mock.On(OnFuncErrorMethodName, 0, ExpectedReturnValue, ExpectedError).Return()
	suite.mock.On(OnMethodErrorMethodName, 0, ExpectedError).Return()
	suite.mock.On(OnPanicMethodName, ExpectedError).Return()
	suite.mock.On(OnTimeoutMethodName, time.Second).Return()
}

func (suite *policyEventTestSuite) TestSingleOnFuncRetryEvent(){
	t := suite.T()
	mock := suite.mock
	suite.p = suite.p.WithOnFuncRetry(func(retriedCount int, returnValue interface{}, err error) {
		mock.OnFuncError(retriedCount, returnValue, err)
	}).(*policy)
	suite.p.onFuncError(0, ExpectedReturnValue, ExpectedError)
	mock.AssertCalled(t, OnFuncErrorMethodName, 0, ExpectedReturnValue, ExpectedError)
	mock.AssertNumberOfCalls(t, OnFuncErrorMethodName, 1)
}

func (suite *policyEventTestSuite) TestMultipleOnFuncRetryEvents(){
	t := suite.T()
	mock := suite.mock
	for i:=0; i < 2; i++ {
		suite.p = suite.p.WithOnFuncRetry(func(retriedCount int, returnValue interface{}, err error) {
			mock.OnFuncError(retriedCount, ExpectedReturnValue, ExpectedError)
		}).(*policy)
	}

	suite.p.onFuncError(0, nil, nil)
	mock.AssertCalled(t, OnFuncErrorMethodName, 0, ExpectedReturnValue, ExpectedError)
	mock.AssertNumberOfCalls(t, OnFuncErrorMethodName, 2)
}

func (suite *policyEventTestSuite) TestSingleOnPanicEvent(){
	suite.p = suite.p.WithOnPanic(func(panicError interface{}) {
		suite.mock.OnPanic(panicError)
	}).(*policy)
	suite.p.onPanic(ExpectedError)
	suite.mock.AssertCalled(suite.T(), OnPanicMethodName, ExpectedError)
	suite.mock.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 1)
}

func (suite *policyEventTestSuite) TestMultipleOnPanicEvents(){
	for i:=0; i < 2; i++ {
		suite.p = suite.p.WithOnPanic(func(panicError interface{}) {
			suite.mock.OnPanic(panicError)
		}).(*policy)
	}
	suite.p.onPanic(ExpectedError)
	suite.mock.AssertCalled(suite.T(), OnPanicMethodName, ExpectedError)
	suite.mock.AssertNumberOfCalls(suite.T(), OnPanicMethodName, 2)
}

func (suite *policyEventTestSuite) TestMultipleOnTimeoutEvents(){
	for i := 0; i < 2; i++ {
		suite.p = suite.p.WithOnTimeout(func(timeout time.Duration) {
			suite.mock.OnTimeout(timeout)
		}).(*policy)
	}
	suite.p.onTimeout(time.Second)
	suite.mock.AssertCalled(suite.T(), OnTimeoutMethodName, time.Second)
	suite.mock.AssertNumberOfCalls(suite.T(), OnTimeoutMethodName, 2)
}

func (suite *policyEventTestSuite) TestMultipleOnMethodRetryEvents(){
	for i := 0; i < 2; i++ {
		suite.p = suite.p.WithOnMethodRetry(func(retriedCount int, err error) {
			suite.mock.OnMethodError(retriedCount, err)
		}).(*policy)
	}
	suite.p.onMethodError(0, ExpectedError)
	suite.mock.AssertCalled(suite.T(), OnMethodErrorMethodName, 0, ExpectedError)
	suite.mock.AssertNumberOfCalls(suite.T(), OnMethodErrorMethodName, 2)
}