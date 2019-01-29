# Error types (WIP)

**[You can find all the code for here](https://github.com/quii/learn-go-with-tests/tree/master/q-and-a/error-types)**

Pedro on the Gopher Slack asks

> If I’m creating an error like `fmt.Errorf("%s must be foo, got %s", bar, baz)`, is there a way to test equality without comparing the string value?

Let's make up a function to help explore this idea. 

```go
// DumbGetter will get the string body of url if it gets a 200
func DumbGetter(url string) (string, error) {
	res, err := http.Get(url)

	if err != nil {
		return "", fmt.Errorf("problem fetching from %s, %v", url, err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("did not get 200 from %s, got %d", url, res.StatusCode)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body) // ignoring err for brevity

	return string(body), nil
}
```

It's not uncommon to write a function that might fail for different reasons and we want to make sure we handle each scenario correctly.

As Pedro says, we _could_ write a test for the status error like so.

```go
t.Run("when you dont get a 200 you get a status error", func(t *testing.T) {

    svr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        res.WriteHeader(http.StatusTeapot)
    }))
    defer svr.Close()

    _, err := DumbGetter(svr.URL)

    if err == nil {
        t.Fatal("expected an error")
    }

    want := fmt.Sprintf("did not get 200 from %s, got %d", svr.URL, http.StatusTeapot)
    got := err.Error()

    if got != want {
        t.Errorf(`got "%v", want "%v"`, got, want)
    }
})
```

This test creates a server which always returns `StatusTeapot` and then we use its URL as the argument to `DumbGetter` so we can see it handles non `200` responses correctly.

This book tries to emphasise _listen to your tests_. 

This test doesn't feel good:

- We're constructing the same string as production code does to test it
- It's annoying to read and write

What does this tell us? The ergonomics of our test would be reflected on another bit of code trying to use our code. How does a user of our code react to the specific kind of errors we return? The best they can do is look at the error string which is extremely error prone and horrible to write.

When writing tests from a TDD approach we have the benefit of getting into the mindset of:

> How would _I_ want to use this code? 

What we could do for `DumbGet` is provide a way for users to use the type system to understand what kind of error has happened. 

As discussed, let's start with a test.

```go
t.Run("when you dont get a 200 you get a status error", func(t *testing.T) {

    svr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        res.WriteHeader(http.StatusTeapot)
    }))
    defer svr.Close()

    _, err := DumbGetter(svr.URL)

    if err == nil {
        t.Fatal("expected an error")
    }

    got, isStatusErr := err.(BadStatusError)

    if !isStatusErr {
        t.Fatalf("was not a BadStatusError, got %T", err)
    }

    want := BadStatusError{URL:svr.URL, Status:http.StatusTeapot}

    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
})
```

We've added the type `BadStatusError` which implements the error interface

```go
type BadStatusError struct {
	URL    string
	Status int
}

func (b BadStatusError) Error() string {
	return fmt.Sprintf("did not get 200 from %s, got %d", b.URL, b.Status)
}
```

When we run the test, it tells us we didn't return the right kind of error

```
--- FAIL: TestDumbGetter (0.00s)
    --- FAIL: TestDumbGetter/when_you_dont_get_a_200_you_get_a_status_error (0.00s)
    	error-types_test.go:56: was not a BadStatusError, got *errors.errorString
```

So let's fix `DumbGet` by updating our error handling code to use our type

```go
if res.StatusCode != http.StatusOK {
    return "", BadStatusError{URL: url, Status: res.StatusCode}
}
```

This change has had some _real positive effects_

- Our `DumbGet` function has become simper to write, it's no longer concerned with the intricacies of the error string, it just creates a `BadStatusError`
- Our tests now reflect what a user of our code _could_ do if they decided they wanted to do some more sophisticated error handling than just logging. Just do a type assertion and then you get easy access to the properties of the error. 
 
