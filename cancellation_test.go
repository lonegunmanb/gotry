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
	cancellation := NewCancellation()
	assert.False(t, cancellation.IsCancellationRequested())
	cancelRequested := cancellation.Cancel()
	assert.True(t, cancelRequested)
	assert.True(t, cancellation.IsCancellationRequested())
}

func TestTwiceCancel(t *testing.T){
	cancellation := NewCancellation()
	_ = cancellation.Cancel()
	cancelRequested := cancellation.Cancel()
	assert.False(t, cancelRequested)
	assert.True(t, cancellation.IsCancellationRequested())
}

func TestOnCancel(t *testing.T){
	cancellation, mockCancel := buildOnCancelMock(NewCancellation())
	cancellation.Cancel()
	mockCancel.AssertCalled(t, onCancelName)
}

func TestMultipleOnCancelEvents(t *testing.T){
	cancellation, cancelHook := buildOnCancelMock(NewCancellation())
	anotherCancelHook := &mockCancel{}
	anotherCancelHook.On(onCancelName).Return()
	cancellation = cancellation.OnCancel(func(){anotherCancelHook.OnCancel()})
	cancellation.Cancel()
	cancelHook.AssertCalled(t, onCancelName)
	anotherCancelHook.AssertCalled(t, onCancelName)
}

func TestCompositeCancellation_One_Way(t *testing.T){
	_, mockCancel1, cancellation2, mockCancel2 := build_bi_way_compoiste_cancellations()
	cancellation2.Cancel()
	mockCancel1.AssertCalled(t, onCancelName)
	mockCancel2.AssertCalled(t, onCancelName)
}

func TestCompositeCancellation_No_Back_Way(t *testing.T){
	cancellation1, mockCancel1, _, mockCancel2 := build_bi_way_compoiste_cancellations()
	cancellation1.Cancel()
	mockCancel1.AssertCalled(t, onCancelName)
	mockCancel2.AssertNotCalled(t, onCancelName)
}

func build_bi_way_compoiste_cancellations() (Cancellation, *mockCancel, Cancellation, *mockCancel) {
	cancellation1, cancelHook1 := buildOnCancelMock(NewCancellation())
	cancellation2 := newCompositeCancellation(cancellation1)
	cancelHook2 := &mockCancel{}
	cancelHook2.On(onCancelName).Return()
	cancellation2 = cancellation2.OnCancel(func() {cancelHook2.OnCancel()})
	return cancellation1, cancelHook1, cancellation2, cancelHook2
}

func buildOnCancelMock(cancellation Cancellation) (Cancellation, *mockCancel){
	cancelHook := &mockCancel{}
	cancelHook.On(onCancelName).Return()
	cancellation = cancellation.OnCancel(func(){cancelHook.OnCancel()})
	return cancellation, cancelHook
}