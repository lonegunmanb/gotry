package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCancel struct {
	mock.Mock
}

func (mockCancel *mockCancel) OnCancel(){
	mockCancel.Called()
}

func TestCancel(t *testing.T){
	cancellation := &cancellation{}
	assert.False(t, cancellation.IsCancellationRequested())
	cancelRequested := cancellation.Cancel()
	assert.True(t, cancelRequested)
	assert.True(t, cancellation.IsCancellationRequested())
}

func TestTwiceCancel(t *testing.T){
	cancellation := &cancellation{}
	_ = cancellation.Cancel()
	cancelRequested := cancellation.Cancel()
	assert.False(t, cancelRequested)
	assert.True(t, cancellation.IsCancellationRequested())
}

func TestOnCancel(t *testing.T){
	cancellation := &cancellation{}
	cancelHook := &mockCancel{}
	cancelHook.On("OnCancel").Return()
	cancellation.OnCancel(func(){cancelHook.OnCancel()})
	cancellation.Cancel()
	cancelHook.AssertCalled(t, "OnCancel")
}