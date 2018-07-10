package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCancel struct {
	mock.Mock
}
const onCancelName = "OnCancel"
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
	cancellation, mockCancel := BuildOnCancelMock(&cancellation{})
	cancellation.Cancel()
	mockCancel.AssertCalled(t, onCancelName)
}

func TestMultipleOnCancelEvents(t *testing.T){
	cancellation, cancelHook := BuildOnCancelMock(&cancellation{})
	anotherCancelHook := &mockCancel{}

	anotherCancelHook.On(onCancelName).Return()
	cancellation = cancellation.OnCancel(func(){anotherCancelHook.OnCancel()})
	cancellation.Cancel()
	cancelHook.AssertCalled(t, onCancelName)
	anotherCancelHook.AssertCalled(t, onCancelName)
}

func BuildOnCancelMock(cancellation Cancellation) (Cancellation, *mockCancel){
	cancelHook := &mockCancel{}
	cancelHook.On(onCancelName).Return()
	cancellation = cancellation.OnCancel(func(){cancelHook.OnCancel()})
	return cancellation, cancelHook
}