package gotry

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

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
