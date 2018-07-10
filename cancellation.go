package gotry

import "sync/atomic"

const cancellationNotRequestedFlag = 0
const cancellationRequestedFlag = 1
type Cancellation interface {
	IsCancellationRequested() bool
	Cancel() bool
	OnCancel(onCancel OnCancel) Cancellation
}
type OnCancel func()

type cancellation struct {
	isCancellationRequestedFlag int32
	onCancel OnCancel
}

func NewCancellation() Cancellation{
	return &cancellation{}
}

func newCompositeCancellation(c Cancellation) Cancellation{
	var newCancellation Cancellation = &cancellation{}
	newCancellation = newCancellation.OnCancel(func(){
		c.Cancel()
	})
	return newCancellation
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
func (cancellation cancellation) OnCancel(onCancel OnCancel) Cancellation {
	originOnCancel := cancellation.onCancel
	if originOnCancel == nil {
		cancellation.onCancel = onCancel
	} else {
		cancellation.onCancel = func() {
			originOnCancel()
			onCancel()
		}
	}
	return &cancellation
}
