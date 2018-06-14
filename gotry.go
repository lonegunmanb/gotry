package gotry


type Func func() (returnValue interface{}, isReturnValueValid bool, err error)
type OnRetryFunc func(retryCount int, returnValue interface{}, err error)
type Method func() (error)
type OnRetryMethod func(retryCount int, err error)

type Policy struct{
	retryLimit int
}

func (policy *Policy) SetRetry(retryLimit int) {
	policy.retryLimit = retryLimit
}
func (policy *Policy) ExecuteFunc(funcBody Func) (interface{}, bool, error) {
	return policy.ExecuteFuncWithRetryHook(funcBody, nil)
}

func(policy *Policy) ExecuteFuncWithRetryHook(funcBody Func, onRetry OnRetryFunc) (returnValue interface{}, isReturnValueValid bool, err error) {
	for i := 0; i < policy.retryLimit; i++ {
		returnValue, isReturnValueValid, err = funcBody()
		if err == nil && isReturnValueValid {
			return returnValue, true, nil
		}
		if onRetry != nil {
			onRetry(i+1, returnValue, err)
		}
	}
	return
}

func (policy *Policy) ExecuteMethod(methodBody Method) error {
	return policy.ExecuteMethodWithRetryHook(methodBody, nil)
}

func(policy *Policy) ExecuteMethodWithRetryHook(methodBody Method, onRetry OnRetryMethod) (err error) {
	for i := 0; i < policy.retryLimit; i++ {
		err = methodBody()
		if err == nil {
			return nil
		}
		if onRetry != nil {
			onRetry(i+1, err)
		}
	}
	return err
}