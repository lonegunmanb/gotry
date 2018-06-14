package gotry


type Func func() (interface{}, bool, error)
type OnRetryFunc func(interface{}, error)
type Method func() (error)

type Policy struct{
	retryLimit int
}

func (policy *Policy) SetRetry(retryLimit int) {
	policy.retryLimit = retryLimit
}
func (policy *Policy) Execute(funcBody Func) (interface{}, bool, error) {
	return policy.ExecuteWithRetryHook(funcBody, nil)
}

func(policy *Policy) ExecuteWithRetryHook(funcBody Func, onRetry OnRetryFunc) (returnValue interface{}, isReturnValueValid bool, err error) {
	for i := policy.retryLimit; i > 0; i-- {
		returnValue, isReturnValueValid, err = funcBody()
		if err == nil && isReturnValueValid {
			return returnValue, true, nil
		}
		if onRetry != nil {
			onRetry(returnValue, err)
		}
	}
	return
}