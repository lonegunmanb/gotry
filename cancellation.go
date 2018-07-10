package gotry

import "sync/atomic"

const cancellationNotRequestedFlag = 0
const cancellationRequestedFlag = 1
type Cancellation interface {
	IsCancellationRequested() bool
	Cancel() bool
}
type OnCancel func()

type cancellation struct {
	isCancellationRequestedFlag int32
	onCancel OnCancel
	
}
func (cancellation *cancellation) IsCancellationRequested() bool {
	return atomic.LoadInt32(&cancellation.isCancellationRequestedFlag) == cancellationRequestedFlag
}
func (cancellation *cancellation) Cancel() bool {
	if successfulCanceled(cancellation) {
		notifyOnCancel(cancellation)
		return true
	}
	return false
}

func notifyOnCancel(cancellation *cancellation) {
	if cancellation.onCancel != nil{
		cancellation.onCancel()
	}
}

func successfulCanceled(cancellation *cancellation) bool {
	return atomic.CompareAndSwapInt32(&cancellation.isCancellationRequestedFlag,
		cancellationNotRequestedFlag,
		cancellationRequestedFlag)
}
func (cancellation *cancellation) OnCancel(onCancel OnCancel) {
	cancellation.onCancel = onCancel
}
