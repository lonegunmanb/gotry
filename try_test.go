package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type Integer int

const ExpectedReturnValue Integer = 1

func (input Integer) IsSuccess() bool{
	return input == ExpectedReturnValue
}

type TryTestSuite struct {
	suite.Suite
	policy *Policy
}

func (suite *TryTestSuite) SetupTest() {
	suite.policy = &Policy{}
	suite.policy.SetRetry(1)
}

func (suite *TryTestSuite) TestSuccessFunc(){
	var successFunc Func = func() (ReturnValue, error){
		var returnValue = ExpectedReturnValue
		return returnValue, nil
	}
	var returnValue, err = suite.policy.Execute(successFunc)
	assert.Nil(suite.T(), err)
	assert.IsType(suite.T(), ExpectedReturnValue, returnValue)
	assert.Equal(suite.T(), ExpectedReturnValue, returnValue, "should be equal")
}

func TestTrySuite(t *testing.T) {
	suite.Run(t, &TryTestSuite{})
}



