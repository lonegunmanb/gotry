# Gotry

Gotry is a golang resilience and transient-error-handling tool that allows developers to express Retry policies. The idea is from [Polly](https://github.com/App-vNext/Polly), so developers can define Retry policy once and use everywhere that function call may encounter transient error.

Gotry is thread-safe, so feel free to use one policy in different goroutines concurrently.

Gotry is written under go1.10.3

**THIS PROJECT IS IN ALPHA SO PLEASE DO NOT USE IN PRODUCTION.

# Usage NewPolicy
```golang
policy := NewPolicy()
```
Default new policy WILL NOT retry.

# Usage Set Retry Limit
```golang
policy := NewPolicy().WithRetryLimit(1)
```
This policy will retry once on error/invalid return/panic

# Usage Retry Forever
```golang
policy := NewPolicy().WithRetryForever()
```
This policy will keep trying until success

# Usage Retry By Custom Predicate
```golang
stopRetry := func(retried int) bool{
    return retried > 3
}
policy := NewPolicy().WithRetryUntil(stopRetry)
```
This policy will keep trying until success or stopRetry return true

# Usage Retry With Timeout
```golang
policy := NewPolicy().WithTimeout(time.Second)
```
This policy will keep trying within one second.

# Usage LetItPanic
```golang
policy := NewPolicy().WithLetItPanic()
```
This policy WILL NOT retry if panic occured. Policy WILL treat panic as error by default.

# Func And Method
Func return FuncReturn
```golang
type FuncReturn struct {
	ReturnValue interface{}
	Valid       bool
	Err         error
}
type Func func() FuncReturn
```
ReturnValue is return by function you tried. Valid is whether return value is valid. Err is error funcation returned.

Method return error. Invoke method return either success or error or panic.
```golang
type Method func() error
```

# Usage Try Function
```golang
policy.TryFunc(func() FuncReturn{
    ...
    result, err := db.Exec("UPDATE .....")
    return FuncReturn{ReturnValue:result, Valid:result.RowsAffected == 1, Err:err}
})
```
Policy'll retry if Valid is false or Err is not nil or panic occured

# Usage Try Method
```golang
policy.TryMethod(func() error {
    ...
    err := db.Ping()
    return err
})
```
Policy'll retry if err is not nil or panic occured

# Usage Retry With Cancellation
```golang
c := NewCancellation()
policy := NewPolicy().WithRetryForever()

go policy.TryMethodWithCancellation(func() error {
    ...//Some code that will never success
}, c)
...
c.Cancel()//policy will stop trying ASAP
```

# Usage OnFuncRetry
```golang
type OnFuncError func(retriedCount int, returnValue interface{}, err error)
policy = policy.WithOnFuncRetry(func(retriedCount int, returnValue interface{}, err error){
    //policy will call this event BEFORE retry func
})
```

# Usage OnMethodRetry
```golang
type OnMethodError func(retriedCount int, err error)
policy = policy.WithOnMethodRetry(func(retriedCount int, err error){
    //policy will call this event BEFORE retry method
})
```

# Usage OnPanic
```golang
type OnPanic func(panicError interface{})
policy = policy.WithOnPanic(func(panicError interface{}){
    //policy will call this event AFTER panic
})
```
OnPanic will be fired even you set LetItPanic(). LetItPanic() will just disable retry, not OnPanic event.

# Usage OnTimeout
```golang
type OnTimeout func(timeout time.Duration)
policy = policy.WithOnTimeout(func(timeout time.Duration){
    //policy will call this event AFTER timeout
    //timeout is THE TIMEOUT you've set on policy.
})
```

# License
Licensed under terms of Apache License Version 2.0