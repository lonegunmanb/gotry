package gotry

type ReturnValue interface {
	IsSuccess() bool
}

type Func func() (ReturnValue, error)
type Method func() (error)

type Policy struct{
	retryAttempt int

}

func (policy *Policy) SetRetry(retryAttempt int) {

}
func (policy *Policy) Execute(successFunc Func) (ReturnValue, error) {
	return successFunc()
}
