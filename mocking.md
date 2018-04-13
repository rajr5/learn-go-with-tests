# Mocking

We'll next cover _mocking_ and it's relation to DI with a case-study. 

You have been asked to write a program which will count 3 seconds, printing each number on a new line and when it reaches zero it will print "Go!" and exit. 

```
3
2
1
Go!
```

We'll tackle this by writing a function called `Countdown` which we will then put inside a `main` program so it looks something like this:

```go
package main

func main() {
    Countdown()
}
```

While this is a pretty trivial program, to test it fully we will need as always to take an _iterative_, _test-driven_ approach. 

What do I mean by iterative? We make sure we take the smallest steps we can to have _useful software_. 

We dont want to spend a long time with code that will theoretically work after some hacking because that's often how developers fall down rabit holes. **It's an important skill to be able to slice up requirements as small as you can so you can have _working software_.**

Here's how we can divide our work up and iterate on it

- Print 3 
- Print 3 to Go!
- Wait a second between each line

Let's just work on the first one.

## Write the test first

Our software needs to print to stdout and we saw how we could use DI to facilitate testing this in the DI section.

```go
func TestCountdown(t *testing.T) {
	buffer := &bytes.Buffer{}

	Countdown(buffer)

	got := buffer.String()
	want := "3"

	if got != want {
		t.Errorf("got '%s' want '%s'", got, want)
	}
}
```

If anything like `buffer` is unfamiliar to you, re-read the previous section.

We know we want our `Countdown` function to write data somewhere and `io.Writer` is the de-facto way of capturing that as an interface in Go. 

- In `main` we will send to `os.Stdout` so our users see the countdown printed to the terminal
- In test we will send to `bytes.Buffer` so our tests can capture what data is being generated

## Try and run the test

`./countdown_test.go:11:2: undefined: Countdown`


## Write the minimal amount of code for the test to run and check the failing test output

Define `Countdown`

```go
func Countdown() {}
```

Try again

```go
./countdown_test.go:11:11: too many arguments in call to Countdown
	have (*bytes.Buffer)
    want ()
```

The compiler is telling you what your function signature could be, so update it.

```go
func Countdown(out *bytes.Buffer) {}
```

`countdown_test.go:17: got '' want '3'`

Perfect!

## Write enough code to make it pass

```go
func Countdown(out *bytes.Buffer) {
	fmt.Fprint(out, "3")
}
```

We're using `fmt.Fprint` which takes an `io.Writer` (like `*bytes.Buffer`) and sends a `string` to it. The test should pass. 

## Refactor

We know that while `*bytes.Buffer` works, it would be better to use a general purpose interface instead.

```go
func Countdown(out io.Writer) {
	fmt.Fprint(out, "3")
}
```

Re-run the tests and they should be passing. 

To complete matters, let's now wire up our function into a `main` so we have some working software to reassure ourselves we're making progress.

```go
package main

import (
	"fmt"
	"io"
	"os"
)

func Countdown(out io.Writer) {
	fmt.Fprint(out, "3")
}

func main() {
	Countdown(os.Stdout)
}
```

Try and run the program and be amazed at your handywork. 

Yes this seems trivial but this approach is what I would recommend for any project. **Take a thin slice of functionality and make it work end-to-end, backed by tests.**

Next we can make it print 4,3,2,1 and then "Go!".

## Write the test first

```go
func TestCountdown(t *testing.T) {
	buffer := &bytes.Buffer{}

	Countdown(buffer)

	got := buffer.String()
	want := `3
2
1
Go!`

	if got != want {
		t.Errorf("got '%s' want '%s'", got, want)
	}
}
```

The backtick syntax is another way of creating a `string` but lets you put things like newlines which is perfect for our test.

Isn't it nice that by focusing on getting our software working end-to-end that iterating on the requirements is very straightforward. By investing in getting the overall plumbing working right, we can focus on the next requirements easily. 

## Try and run the test

```
countdown_test.go:21: got '3' want '3
		2
        1
		Go!'
```
## Write enough code to make it pass

```go
func Countdown(out io.Writer) {
	for i := 3; i > 0; i-- {
		fmt.Fprintln(out, i)
    }
    fmt.Fprint(out, "Go!")
}
```

Use a `for` loop counting backwards with `i--` and use `fmt.Fprintln` to print to `out` with our number followed by a newline character. Finally use `fmt.Fprint` to send "Go!" aftward

## Refactor

There's not much to refactor other than removing some magic values.

```go
const finalWord = "Go!"
const countdownStart = 3

func Countdown(out io.Writer) {
	for i := countdownStart; i > 0; i-- {
		fmt.Fprintln(out, i)
	}
	fmt.Fprint(out, finalWord)
}
```

If you run the program now, you should get the desired output but we dont have it as a dramatic countdown with the 1 second pauses. 

Go let's you achieve this with `time.Sleep`. Try adding it in to our code.

```go
func Countdown(out io.Writer) {
	for i := countdownStart; i > 0; i-- {
		time.Sleep(1 * time.Second)
		fmt.Fprintln(out, i)
	}
	
	time.Sleep(1 * time.Second)
	fmt.Fprint(out, finalWord)
}
```

If you run the program it works as we want it to.

## Mocking

The tests still pass and the software works as intended but we have some problems:
- Our tests take 6 seconds to run. Every forward thinking post about software development emphasises the importance of quick feedback loops. **Slow tests ruin developer productivity**.
- We have not tested an important property of our function. 

We have a dependency on `Sleep`ing which we need to extract so we can then control it in our tests.

We want to assert that after every count we `Sleep` for a second.

If we can _mock_ `time.Sleep` we can use _dependency injection_ to use it instead of a "real" `time.Sleep` and then we can **spy on the calls** to make assertions on them. 

## Write the test first

Let's define our dependency

```go
type Sleeper interface {
	Sleep()
}
```

I made a design decision that our `Countdown` function would not be responsible
for how long the sleep is. This simplifies our code a little for now at least
and means a user of our function can configure that sleepiness however they
like.

Now we need to make a _mock_ of it for our tests to use. 

```go
type SpySleeper struct {
	Calls int
}

func (s *SpySleeper) Sleep() {
	s.Calls++
}
```

There's nothing new to learn here, we just need to be able to _spy_ on the calls to our mock so we can check it has been called `N` times.

Update the tests to inject a dependency on our Spy and assert that the sleep has been called 6 times.

```go
func TestCountdown(t *testing.T) {
	buffer := &bytes.Buffer{}
	spySleeper := &SpySleeper{}

	Countdown(buffer, spySleeper)

	got := buffer.String()
	want := `3
2
1
Go!`

	if got != want {
		t.Errorf("got '%s' want '%s'", got, want)
	}

	if spySleeper.Calls != 4 {
		t.Errorf("not enough calls to sleeper, want 4 got %d", spySleeper.Calls)
	}
}
```

## Try and run the test

```
too many arguments in call to Countdown
	have (*bytes.Buffer, Sleeper)
	want (io.Writer)
```

## Write the minimal amount of code for the test to run and check the failing test output

We need to update `Countdown` to accept our `Sleeper`

```go
func Countdown(out io.Writer, sleeper Sleeper) {
	for i := countdownStart; i > 0; i-- {
		time.Sleep(1 * time.Second)
		fmt.Fprintln(out, i)
	}

	time.Sleep(1 * time.Second)
	fmt.Fprint(out, finalWord)
}
```

If you try again, your `main` will no longer compile for the same reason

```
./main.go:26:11: not enough arguments in call to Countdown
	have (*os.File)
	want (io.Writer, Sleeper)
```

Let's create a _real_ sleeper which implements the interface we need

```go
type ConfigurableSleeper struct {
	duration time.Duration
}

func (o *ConfigurableSleeper) Sleep() {
	time.Sleep(o.duration)
}
```

I decided to make a little extra effort and make it so our real sleeper is
configurable but you could just as easily not bother and hard-code it for
1 second (YAGNI right?).

We can then use it in our real application like so

```go
func main() {
	sleeper := &ConfigurableSleeper{1 * time.Second}
	Countdown(os.Stdout, sleeper)
}
```

## Write enough code to make it pass

The test is now compiling but not passing because we're still calling the `time.Sleep` rather than the injected in dependency. Let's fix that.

```go
func Countdown(out io.Writer, sleeper Sleeper) {
	for i := countdownStart; i > 0; i-- {
		sleeper.sleep()
		fmt.Fprintln(out, i)
	}

	sleeper.sleep()
	fmt.Fprint(out, finalWord)
}
```

Now the test should be passing (and no longer taking 6 seconds!).

### Still some problems

There's still another important property we haven't tested. 

The important thing about the function is that it sleeps before the first print and then after each one until the last, e.g:

- `Sleep`
- `Print N`
- `Sleep`
- `Print N-1`
- `Sleep`
- etc

Our latest change only asserts that it has slept 4 times, but those sleeps could occur out of sequence

When writing tests and you're not confident you have working code, just break it! Change the code to the following

```go
func Countdown(out io.Writer, sleeper Sleeper) {
	for i := countdownStart; i > 0; i-- {
		sleeper.Sleep()
	}

	for i := countdownStart; i > 0; i-- {
		fmt.Fprintln(out, i)
	}

	sleeper.Sleep()
	fmt.Fprint(out, finalWord)
}
```

If you run your tests they should still be passing.

Let's use spying again with a new test to check the order of operations is correct.

We have two different dependencies and we want to record all of their operations into one list. So we'll create _one Spy for them both_.

```go
type CountdownOperationsSpy struct {
	Calls []string
}

func (s *CountdownOperationsSpy) Sleep() {
	s.Calls = append(s.Calls, sleep)
}

func (s *CountdownOperationsSpy) Write(p []byte) (n int, err error) {
	s.Calls = append(s.Calls, write)
	return
}

const write = "write"
const sleep = "sleep"
```

Our `CountdownOperationsSpy` implements both types of dependencies and records every call into one slice. In this test we're only concerned about the order of operations, so just recording them as list of named operations is sufficient. 

We can now add a sub-test into our test suite.


```go
t.Run("sleep after every print", func(t *testing.T) {
    spySleepPrinter := &CountdownOperationsSpy{}
    Countdown(spySleepPrinter, spySleepPrinter)

    want := []string{
    	sleep,
        write,
        sleep,
        write,
        sleep,
        write,
        sleep,
        write,
    }

    if !reflect.DeepEqual(want, spySleepPrinter.Calls) {
        t.Errorf("wanted calls %v got %v", want, spySleepPrinter.Calls)
    }
})
```

This test should now fail. Revert it back and the new test should pass. 

We now have two tests spying on the `Sleeper` so we can now refactor our test so one is testing what is being printed and the other one is ensuring we're sleeping in between the prints. Finally we can delete our first spy as it's not used anymore. 

```go
func TestCountdown(t *testing.T) {

	t.Run("prints 3 to Go!", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		Countdown(buffer, &CountdownOperationsSpy{})

		got := buffer.String()
		want := `3
2
1
Go!`

		if got != want {
			t.Errorf("got '%s' want '%s'", got, want)
		}
	})

	t.Run("sleep after every print", func(t *testing.T) {
		spySleepPrinter := &CountdownOperationsSpy{}
		Countdown(spySleepPrinter, spySleepPrinter)

		want := []string{
			sleep,
			write,
			sleep,
			write,
			sleep,
			write,
			sleep,
			write,
		}

		if !reflect.DeepEqual(want, spySleepPrinter.Calls) {
			t.Errorf("wanted calls %v got %v", want, spySleepPrinter.Calls)
		}
	})
}
```

Finally we have our function and it's two important properties properly tested.

## But isn't mocking evil?

You may have heard mocking is evil. Just like anything in software development it can be used for evil, just like DRY. 

People normally get in to a bad state when they don't _listen to their tests_ and _not respecting the refactoring stage_. 

If your mocking code is becoming complicated or you are having to mock out lots of things to test something, you should _listen_ to that bad feeling and think about your code. Usually it is a sign of

- The thing you are testing is having to do too many things. 
- Its dependencies are too fine-grained
- Your test is too concerned with implementation details

Normally a lot of mocking points to _bad abstraction_ in your code. 

**What people see here is a weakness in TDD but it is actually a strength**, more often than not poor test code is a result of bad design or put more nicely, well-designed code is easy to test. 

### But mocks and tests are still making my life hard! 

Ever run into this situation?

- You want to do some refactoring
- To do this you have to end up changing lots of tests and lots of mocks
- You question TDD and make a post on Medium titled "Mocking considered harmful"

This is usually a sign of you testing too much _implementation detail_. Try to make it so your tests are testing _useful behaviour_ unless the implementation is really important to how the system runs.

It is sometimes hard to know _what level_ to test exactly but here are some thought processes and rules I try to follow

- **The definition of refactoring is that the code changes but the behaviour stays the same**. If you have decided to do some refactoring in theory you should be able to do make the commit without any test changes. So when writing a test ask yourself
    - Am i testing the behaviour I want or the implementation details?
    - If i were to refactor this code, would I have to make lots of changes to the tests?
- Although Go lets you test private functions, I would avoid it as private functions are to do with implementation.
- I feel like if a test is working with **more than 3 mocks then it is a red flag** - time for a rethink on the design
- Use spies with caution. Spies let you see the insides of the algorithm you are writing which can be very useful but that means a tighter coupling between your test code and the implementation. **Be sure you actually care about these details if you're going to spy on them**

As always, rules in software development aren't really rules and there can be exceptions. [Uncle Bob's article of "When to mock"](https://8thlight.com/blog/uncle-bob/2014/05/10/WhenToMock.html) has some excellent pointers.

## Wrapping up

- **Without mocking important areas of your code will be untested**. In our case we would not be able to test that our code paused between each print but there are countless other examples. Calling a service that _can_ fail? Wanting to test your system in a particular state? It is very hard to test these scenarios without mocking.
- Without mocks you may have to set up databases and other third parties things just to test simple business rules. You're likely to have slow tests, resulting in **slow feedback loops**.
- By having to spin up a database or a webservice to test something you're likely to have **fragile tests** due to the unreliability of such services.

Once a developer learns about mocking it becomes very easy to over-test every single facet of a system in terms of the _way it works_ rather than _what it does_. Always be mindful about **the value of your tests** and what impact they would have in future refactoring.

In this post about mocking we have only covered **Spies** which are a kind of mock. There are different kind of mocks. [Uncle Bob explains the types in a very easy and short article](https://8thlight.com/blog/uncle-bob/2014/05/14/TheLittleMocker.html), go read it. 
