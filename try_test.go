package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

type Integer int

const ExpectedReturnValue Integer = 1

func (input Integer) IsSuccess() bool{
	return input == ExpectedReturnValue
}

func TestSuccessFunc(t *testing.T){
	policy := &Policy{}
	policy.SetRetry(1)
	var successFunc Func = func() (ReturnValue, error){
		var returnValue Integer = ExpectedReturnValue
		return returnValue, nil
	}
	var returnValue, err = policy.Execute(successFunc)
	assert.Nil(t, err)
	assert.IsType(t, ExpectedReturnValue, returnValue)
	assert.Equal(t, ExpectedReturnValue, returnValue, "should be equal")
}



