package gotry

import "sync/atomic"

const cancellationNotRequestedFlag = 0
const cancellationRequestedFlag = 1
type Cancellation interface {
	IsCancellationRequested() bool
	Cancel() bool
}

type cancellation struct {
	isCancellationRequestedFlag int32
}
func (cancellation *cancellation) IsCancellationRequested() bool {
	return atomic.LoadInt32(&cancellation.isCancellationRequestedFlag) == cancellationRequestedFlag
}
func (cancellation *cancellation) Cancel() bool {
	return atomic.CompareAndSwapInt32(&cancellation.isCancellationRequestedFlag,
													cancellationNotRequestedFlag,
													cancellationRequestedFlag)
}
