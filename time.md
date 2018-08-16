# Time (WIP)

**[You can find all the code for this chapter here](https://github.com/quii/learn-go-with-tests/tree/master/time)**

The product owner wants us to expand the functionality of our command line application by helping a group of people play Texas-Holdem Poker.

## Just enough information on poker

You wont need to know much about poker, only that at certain time intervals all the players need to be informed of a steadily increasing "blind" value. 

Our application will help keep track of when the blind should go up, and how much it should be.

- Create a command line app.
- When it starts it asks how many players are playing. This determines the amount of time there is before the "blind" bet goes up.
  - There is a base amount of time of 5 minutes.
  - For every player, 1 minute is added.
  - e.g 6 players equals 11 minutes for the blind.
- After the blind time expires the game should alert the players the new amount the blind bet is.
- The blind starts at 100 chips, then 200, 400, 600, 1000, 2000 and continue to double until the game ends.
- When the game ends the user should be able to type "Chris wins" and that will record a win for the player in our existing database. This should then exit the program.

## Reminder of the code

In the previous chapter we made our start to the command line application which already accepts a command of `{name} wins`. Here is what the current `CLI` code looks like, but be sure to familiarise yourself with the other code too before starting.

```go
type CLI struct {
	playerStore PlayerStore
	in          *bufio.Scanner
}

func NewCLI(store PlayerStore, in io.Reader) *CLI {
	return &CLI{
		playerStore: store,
		in:          bufio.NewScanner(in),
	}
}

func (cli *CLI) PlayPoker() {
	userInput := cli.readLine()
	cli.playerStore.RecordWin(extractWinner(userInput))
}

func extractWinner(userInput string) string {
	return strings.Replace(userInput, " wins", "", 1)
}

func (cli *CLI) readLine() string {
	cli.in.Scan()
	return cli.in.Text()
}
```


### `time.AfterFunc`

We want to be able to schedule our program to print the blind bet values at certain durations dependant on the number of players.

To limit the scope of what we need to do, we'll forget about the number of players part and just assume there are 5 players so we'll test that _every 10 minutes the new value of the blind bet is printed_.

As usual the standard library has us covered with [`func AfterFunc(d Duration, f func()) *Timer`](https://golang.org/pkg/time/#AfterFunc)

> `AfterFunc` waits for the duration to elapse and then calls f in its own goroutine. It returns a `Timer` that can be used to cancel the call using its Stop method.

When we call `PlayPoker` we'll schedule all of our blind alerts.

Testing this may be a little tricky though. We'll want to verify that each time period is scheduled with the correct blind amount but if you look at the signature of `time.AfterFunc` its second argument is the function it will run. You cannot compare functions in Go so we'd be unable to test what function has been sent in. So we'll need to write some kind of wrapper around `time.AfterFunc` which will take the time to run and the amount to print so we can spy on that.

## Write the test first

Add a new test to our suite

```go
t.Run("it schedules printing of blind values", func(t *testing.T) {
    in := strings.NewReader("Chris wins\n")
    playerStore := &poker.StubPlayerStore{}
    blindAlerter := &SpyBlindAlerter{}

    cli := poker.NewCLI(playerStore, in, blindAlerter)
    cli.PlayPoker()
    
    if len(blindAlerter.alerts) != 1 {
        t.Fatal("expected a blind alert to be scheduled")
    }
})
```

You'll notice we've made a `SpyBlindAlerter` which we are trying to inject into our `CLI` and then checking that after we call `PlayerPoker` that an alert is scheduled.

(Remember we are just going for the simplest scenario first and then we'll iterate.)

Here's the definition of `SpyBlindAlerter`

```go
type SpyBlindAlerter struct {
	alerts []struct{
		scheduledAt time.Duration
		amount int
	}
}

func (s *SpyBlindAlerter) ScheduleAlertAt(duration time.Duration, amount int) {
	s.alerts = append(s.alerts, struct {
		scheduledAt time.Duration
		amount int
	}{duration,  amount})
}

```


## Try to run the test

```
./CLI_test.go:32:27: too many arguments in call to poker.NewCLI
	have (*poker.StubPlayerStore, *strings.Reader, *SpyBlindAlerter)
	want (poker.PlayerStore, io.Reader)
```

## Write the minimal amount of code for the test to run and check the failing test output

We have added a new argument and the compiler is complaining. _Strictly speaking_ the minimal amount of code is to make `NewCLI` accept a `*SpyBlindAlerter` but let's cheat a little and just define the dependency as an interface.

```go
type BlindAlerter interface {
	ScheduleAlertAt(duration time.Duration, amount int)
}
```

And then add it to the constructor

```go
func NewCLI(store PlayerStore, in io.Reader, alerter BlindAlerter) *CLI
```

Your other tests will now fail as they dont have a `BlindAlerter` passed in to `NewCLI`. 

Spying on BlindAlerter is not relevant for the other tests so in the test file add

```go
var dummySpyAlerter = &SpyBlindAlerter{}
```

Then use that in the other tests to fix the compilation problems. By labelling it as a "dummy" it is clear to the reader of the test that it is not important.

The tests should now compile and our new test fails.

```
=== RUN   TestCLI
=== RUN   TestCLI/it_schedules_printing_of_blind_values
--- FAIL: TestCLI (0.00s)
    --- FAIL: TestCLI/it_schedules_printing_of_blind_values (0.00s)
    	CLI_test.go:38: expected a blind alert to be scheduled
```

## Write enough code to make it pass

We'll need to add the `BlindAlerter` as a field on our `CLI` so we can reference it in our `PlayPoker` method.

```go
type CLI struct {
	playerStore PlayerStore
	in          *bufio.Reader
	alerter     BlindAlerter
}

func NewCLI(store PlayerStore, in io.Reader, alerter BlindAlerter) *CLI {
	return &CLI{
		playerStore: store,
		in:          bufio.NewReader(in),
		alerter:     alerter,
	}
}
```

To make the test pass, we can call our `BlindAlerter` with anything we like

```go
func (cli *CLI) PlayPoker() {
	cli.alerter.ScheduleAlertAt(5 * time.Second, 100)
	userInput, _ := cli.in.ReadString('\n')
	cli.playerStore.RecordWin(extractWinner(userInput))
}
```

Next we'll want to check it schedules all the alerts we'd hope for, for 5 players

## Write the test first

```go
	t.Run("it schedules printing of blind values", func(t *testing.T) {
		in := strings.NewReader("Chris wins\n")
		playerStore := &poker.StubPlayerStore{}
		blindAlerter := &SpyBlindAlerter{}

		cli := poker.NewCLI(playerStore, in, blindAlerter)
		cli.PlayPoker()

		cases := []struct{
			expectedScheduleTime time.Duration
			expectedAmount       int
		} {
			{0 * time.Second, 100},
			{10 * time.Minute, 200},
			{20 * time.Minute, 300},
			{30 * time.Minute, 400},
			{40 * time.Minute, 500},
			{50 * time.Minute, 600},
			{60 * time.Minute, 800},
			{70 * time.Minute, 1000},
			{80 * time.Minute, 2000},
			{90 * time.Minute, 4000},
			{100 * time.Minute, 8000},
		}

		for i, c := range cases {
			t.Run(fmt.Sprintf("%d scheduled for %v", c.expectedAmount, c.expectedScheduleTime), func(t *testing.T) {

				if len(blindAlerter.alerts) <= i {
					t.Fatalf("alert %d was not scheduled %v", i, blindAlerter.alerts)
				}

				alert := blindAlerter.alerts[i]

				amountGot := alert.amount
				if amountGot != c.expectedAmount {
					t.Errorf("got amount %d, want %d", amountGot, c.expectedAmount)
				}

				gotScheduledTime := alert.scheduledAt
				if gotScheduledTime != c.expectedScheduleTime {
					t.Errorf("got scheduled time of %v, want %v", gotScheduledTime, c.expectedScheduleTime)
				}
			})
		}
	})
```

Table-based test works nicely here and clearly illustrate what our requirements are. We run through the table and check the `SpyBlindAlerter` to see if the alert has been scheduled with the correct values.

## Try to run the test

You should have a lot of failures looking like this

```go
=== RUN   TestCLI
--- FAIL: TestCLI (0.00s)
=== RUN   TestCLI/it_schedules_printing_of_blind_values
    --- FAIL: TestCLI/it_schedules_printing_of_blind_values (0.00s)
=== RUN   TestCLI/it_schedules_printing_of_blind_values/100_scheduled_for_0s
        --- FAIL: TestCLI/it_schedules_printing_of_blind_values/100_scheduled_for_0s (0.00s)
        	CLI_test.go:71: got scheduled time of 5s, want 0s
=== RUN   TestCLI/it_schedules_printing_of_blind_values/200_scheduled_for_10m0s
        --- FAIL: TestCLI/it_schedules_printing_of_blind_values/200_scheduled_for_10m0s (0.00s)
        	CLI_test.go:59: alert 1 was not scheduled [{5000000000 100}]
```

## Write enough code to make it pass

```go
func (cli *CLI) PlayPoker() {

	blinds := []int{100, 200, 300, 400, 500, 600, 800, 1000, 2000, 4000, 8000}
	blindTime := 0 * time.Second
	for _, blind := range blinds {
		cli.alerter.ScheduleAlertAt(blindTime, blind)
		blindTime = blindTime + 10*time.Minute
	}

	userInput, _ := cli.in.ReadString('\n')
	cli.playerStore.RecordWin(extractWinner(userInput))
}
```

It's not a lot more complicated than what we already had. We're just now iterating over an array of `blinds` and calling the scheduler on an increasing `blindTime`

## Refactor

We can encapsulate our scheduled alerts into a method just to make `PlayPoker` read a little clearer.

```go
func (cli *CLI) PlayPoker() {
	cli.scheduleBlindAlerts()
	userInput, _ := cli.in.ReadString('\n')
	cli.playerStore.RecordWin(extractWinner(userInput))
}

func (cli *CLI) scheduleBlindAlerts() {
	blinds := []int{100, 200, 300, 400, 500, 600, 800, 1000, 2000, 4000, 8000}
	blindTime := 0 * time.Second
	for _, blind := range blinds {
		cli.alerter.ScheduleAlertAt(blindTime, blind)
		blindTime = blindTime + 10*time.Minute
	}
}
```

Finally our tests are looking a little clunky. We have two anonymous structs representing the same thing, a `ScheduledAlert`. Let's refactor that into a new type and then make some helpers to compare them.

```go
type scheduledAlert struct {
	at time.Duration
	amount int
}

func (s scheduledAlert) String() string {
	return fmt.Sprintf("%d chips at %v", s.amount, s.at)
}

type SpyBlindAlerter struct {
	alerts []scheduledAlert
}

func (s *SpyBlindAlerter) ScheduleAlertAt(at time.Duration, amount int) {
	s.alerts = append(s.alerts, scheduledAlert{at, amount})
}
```

We've added a `String()` method to our type so it prints nicely if the test fails

Update our test to use our new type

```go
t.Run("it schedules printing of blind values", func(t *testing.T) {
    in := strings.NewReader("Chris wins\n")
    playerStore := &poker.StubPlayerStore{}
    blindAlerter := &SpyBlindAlerter{}

    cli := poker.NewCLI(playerStore, in, blindAlerter)
    cli.PlayPoker()

    cases := []scheduledAlert {
        {0 * time.Second, 100},
        {10 * time.Minute, 200},
        {20 * time.Minute, 300},
        {30 * time.Minute, 400},
        {40 * time.Minute, 500},
        {50 * time.Minute, 600},
        {60 * time.Minute, 800},
        {70 * time.Minute, 1000},
        {80 * time.Minute, 2000},
        {90 * time.Minute, 4000},
        {100 * time.Minute, 8000},
    }

    for i, want := range cases {
        t.Run(fmt.Sprint(want), func(t *testing.T) {

            if len(blindAlerter.alerts) <= i {
                t.Fatalf("alert %d was not scheduled %v", i, blindAlerter.alerts)
            }

            got := blindAlerter.alerts[i]
            assertScheduledAlert(t, got, want)
        })
    }
})
```

Implement `assertScheduledAlert` yourself.

We've spent a fair amount of time here writing tests and have been somewhat naughty not integrating with our application. Let's address that before we pile on any more requirements.

Try running the app and it wont compile, complaining about not enough args to `NewCLI`.

Let's create an implementation of `BlindAlerter` that we can use in our application.

Create `BlindAlerter.go` and move our `BlindAlerter` interface and add the new things below

```go
package poker

import (
	"time"
	"fmt"
	"os"
)

type BlindAlerter interface {
	ScheduleAlertAt(duration time.Duration, amount int)
}

type BlindAlerterFunc func(duration time.Duration, amount int)

func (a BlindAlerterFunc) ScheduleAlertAt(duration time.Duration, amount int) {
	a(duration, amount)
}

func StdOutAlerter(duration time.Duration, amount int) {
	time.AfterFunc(duration, func() {
		fmt.Fprintf(os.Stdout, "Blind is now %d\n", amount)
	})
}
```

Remember that any _type_ can implement an interface, not just `structs`. If you are making a library that exposes an interface with one function defined it is a common idiom to also expose a `MyInterfaceFunc` type. This type will be a `func` which will also implement your interface. That way users of your interface can convieniently implement your interface with just a function rather than having to create an empty `struct` type.

We then create the function `StdOutAlerter` which has the same signature as the function and just use `time.AfterFunc` to schedule it to print to `os.Stdout`.

Update `main` where we create `NewCLI` to see this in action

```go
game := poker.NewCLI(store, os.Stdin, poker.BlindAlerterFunc(poker.StdOutAlerter))
```

Before running you might want to change the `blindTime` increment in `CLI` to be 10 seconds rather than 10 minutes just so you can see it in action.

You should see it print the blind values as we'd expect every 10 seconds. Notice how you can still type `Shaun wins` into the CLI and it will stop the program how we'd expect.

The game wont always be played with 5 people so we need to prompt the user to enter a number of players before the game starts. 

## Write the test first

We'll want to record what is written to StdOut. We've done this a few times now, we know that `os.Stdout` is an `io.Writer` so we can check what is written if we use dependency injection to pass in a `bytes.Buffer` in our test and see what our code will write.

We don't care about our other collaborators in this test just yet so we've made some dummies in our test file. We should be a little wary that we now have 4 dependencies for `CLI`, that feels like maybe it is starting to have too many responsibilities. Let's live with it for now and see if a refactoring emerges as we add this new functionality.

```go
var dummyBlindAlerter = &SpyBlindAlerter{}
var dummyPlayerStore = &poker.StubPlayerStore{}
var dummyStdIn = &bytes.Buffer{}
var dummyStdOut = &bytes.Buffer{}
```

And here is our new test

```go
t.Run("it prompts the user to enter the number of players", func(t *testing.T) {
    stdout := &bytes.Buffer{}
    cli := poker.NewCLI(dummyPlayerStore, dummyStdIn, stdout, dummyBlindAlerter)
    cli.PlayPoker()

    got :=stdout.String()
    want := "Please enter the number of players: "

    if got != want {
        t.Errorf("got '%s', want '%s'", got, want)
    }
})
```

We pass in what will be `os.Stdout` in `main` and see what is written.

## Try to run the test

```
./CLI_test.go:38:27: too many arguments in call to poker.NewCLI
	have (*poker.StubPlayerStore, *bytes.Buffer, *bytes.Buffer, *SpyBlindAlerter)
	want (poker.PlayerStore, io.Reader, poker.BlindAlerter)
```

## Write the minimal amount of code for the test to run and check the failing test output

We have a new dependency so we'll have to update `NewCLI`

```go
func NewCLI(store PlayerStore, in io.Reader, out io.Writer, alerter BlindAlerter) *CLI
```

Now the _other_ tests will fail to compile because they dont have an `io.Writer` being passed into `NewCLI`. Add `dummyStdout` for the other tests.

The new test should fail like so

```
=== RUN   TestCLI
--- FAIL: TestCLI (0.00s)
=== RUN   TestCLI/it_prompts_the_user_to_enter_the_number_of_players
    --- FAIL: TestCLI/it_prompts_the_user_to_enter_the_number_of_players (0.00s)
    	CLI_test.go:46: got '', want 'Please enter the number of players: '
FAIL
```

## Write enough code to make it pass

We need to add our new dependency to our `CLI` so we can reference it in `PlayPoker`

```go
type CLI struct {
	playerStore PlayerStore
	in          *bufio.Reader
	out         io.Writer
	alerter     BlindAlerter
}

func NewCLI(store PlayerStore, in io.Reader, out io.Writer, alerter BlindAlerter) *CLI {
	return &CLI{
		playerStore: store,
		in:          bufio.NewReader(in),
		out:         out,
		alerter:     alerter,
	}
}
```

Then finally we can write our prompt at the start of the game

```go
func (cli *CLI) PlayPoker() {
	fmt.Fprint(cli.out, "Please enter the number of players: ")
	cli.scheduleBlindAlerts()
	userInput, _ := cli.in.ReadString('\n')
	cli.playerStore.RecordWin(extractWinner(userInput))
}
```

## Refactor

We have a duplicate string for the prompt which we should extract into a constant

```go
const PlayerPrompt = "Please enter the number of players: "
```

Use this in both the test code and `CLI`.

Now we need to send in a number and extract it out. The only way we'll know if it has had the desired effect is by seeing what blind alerts were scheduled.

## Write the test first

```go
t.Run("it prompts the user to enter the number of players", func(t *testing.T) {
    stdout := &bytes.Buffer{}
    in := strings.NewReader("7\n")
    blindAlerter := &SpyBlindAlerter{}

    cli := poker.NewCLI(dummyPlayerStore, in, stdout, blindAlerter)
    cli.PlayPoker()

    got :=stdout.String()
    want := poker.PlayerPrompt

    if got != want {
        t.Errorf("got '%s', want '%s'", got, want)
    }

    cases := []scheduledAlert{
        {0 * time.Second, 100},
        {12 * time.Minute, 200},
        {24 * time.Minute, 300},
        {36 * time.Minute, 400},
    }

    for i, want := range cases {
        t.Run(fmt.Sprint(want), func(t *testing.T) {

            if len(blindAlerter.alerts) <= i {
                t.Fatalf("alert %d was not scheduled %v", i, blindAlerter.alerts)
            }

            got := blindAlerter.alerts[i]
            assertScheduledAlert(t, got, want)
        })
    }
})
```

Ouch! A lot of changes. 

- We remove our dummy for StdIn and instead send in a mocked version representing our user entering 7
- We also remove our dummy on the blind alerter so we can see that the number of players has had an effect on the scheduling
- We test what alerts are scheduled

## Try to run the test

The test should still compile and fail reporting that the scheduled times are wrong because we've hard-coded for the game to be based on having 5 players

```
=== RUN   TestCLI
--- FAIL: TestCLI (0.00s)
=== RUN   TestCLI/it_prompts_the_user_to_enter_the_number_of_players
    --- FAIL: TestCLI/it_prompts_the_user_to_enter_the_number_of_players (0.00s)
=== RUN   TestCLI/it_prompts_the_user_to_enter_the_number_of_players/100_chips_at_0s
        --- PASS: TestCLI/it_prompts_the_user_to_enter_the_number_of_players/100_chips_at_0s (0.00s)
=== RUN   TestCLI/it_prompts_the_user_to_enter_the_number_of_players/200_chips_at_12m0s
```

## Write enough code to make it pass

Remember, we are free to commit whatever sins we need to make this work. Once we have working software we can then work on refactoring the mess we're about to make!


```go
func (cli *CLI) PlayPoker() {
	fmt.Fprint(cli.out, PlayerPrompt)
	
	numberOfPlayersInput, _ := cli.in.ReadString('\n')
	numberOfPlayers, _ := strconv.Atoi(strings.Trim(numberOfPlayersInput, "\n"))

	cli.scheduleBlindAlerts(numberOfPlayers)

	userInput, _ := cli.in.ReadString('\n')
	cli.playerStore.RecordWin(extractWinner(userInput))
}

func (cli *CLI) scheduleBlindAlerts(numberOfPlayers int) {
	blindIncrement := time.Duration(5+numberOfPlayers) * time.Minute

	blinds := []int{100, 200, 300, 400, 500, 600, 800, 1000, 2000, 4000, 8000}
	blindTime := 0 * time.Second
	for _, blind := range blinds {
		cli.alerter.ScheduleAlertAt(blindTime, blind)
		blindTime = blindTime + blindIncrement
	}
}
```

- We read in the `numberOfPlayersInput` into a string
- We `Trim` the string of the newline entered and use `Atoi` to convert it into an integer - ignoring any error scenarios. We'll need to write a test for that scenario later.
- From here we change `scheduleBlindAlerts` to accept a number of players. We then calculate a `blindIncrement` time to use to add to `blindTime` as we iterate over the blind amounts

While our new test has been fixed, a lot of others have failed because now our system only works if the game starts with a user entering a number. You'll need to fix the tests by changing the user inputs so that a number followed by a newline is added (this is highlighting yet more flaws in our approach right now).

## Refactor

Let's _listen to our tests_. 

- In order to test that we are scheduling some alerts we set up 4 different dependencies. Whenever you have a lot of dependencies for a _thing_ in your system, it implies it's doing too much. Visually we can see it in how cluttered our test is.
- To me it feels like **we need to make a cleaner abstraction between reading user input and the business logic we want to do** 
- A better test would be _given this user input, do we call `Game` with the correct number of players_. 
- We would then extract the testing of the scheduling into the tests for our new `Game`.

We can refactor our `Game` first and our test should continue to pass. Once we've made the structural changes we want we can think about how we can refactor the tests to reflect our new separation of concerns

Remember when making changes in refactoring try to keep them as small as possible and keep re-running the tests.

Try it yourself first. Think about the boundaries of what a `Game` would offer and what our `CLI` should be doing. 

For now **don't** change the external interface of `NewCLI` as we don't want to change the test code and the client code at the same time as that is too much to juggle and we could end up breaking things.

This is what I came up with:

```go
// game.go
type Game struct {
	alerter BlindAlerter
	store   PlayerStore
}

func (p *Game) Start(numberOfPlayers int) {
	blindIncrement := time.Duration(5+numberOfPlayers) * time.Minute

	blinds := []int{100, 200, 300, 400, 500, 600, 800, 1000, 2000, 4000, 8000}
	blindTime := 0 * time.Second
	for _, blind := range blinds {
		p.alerter.ScheduleAlertAt(blindTime, blind)
		blindTime = blindTime + blindIncrement
	}
}

func (p *Game) Finish(winner string) {
	p.store.RecordWin(winner)
}

// cli.go
type CLI struct {
	playerStore PlayerStore
	in          *bufio.Reader
	out         io.Writer
	game        *Game
}

func NewCLI(store PlayerStore, in io.Reader, out io.Writer, alerter BlindAlerter) *CLI {
	return &CLI{
		in:  bufio.NewReader(in),
		out: out,
		game: &Game{
			alerter: alerter,
			store:   store,
		},
	}
}

const PlayerPrompt = "Please enter the number of players: "

// PlayPoker starts the game
func (cli *CLI) PlayPoker() {
	fmt.Fprint(cli.out, PlayerPrompt)

	numberOfPlayersInput, _ := cli.in.ReadString('\n')
	numberOfPlayers, _ := strconv.Atoi(strings.Trim(numberOfPlayersInput, "\n"))

	cli.game.Start(numberOfPlayers)

	winnerInput, _ := cli.in.ReadString('\n')
	winner := extractWinner(winnerInput)

	cli.game.Finish(winner)
}

func extractWinner(userInput string) string {
	return strings.Replace(userInput, " wins\n", "", 1)
}
```

From a "domain" perspective:
- We want to `Start` a `Game`, indicating how many people are playing
- We want to `Finish` a `Game`, declaring the winner

The new `Game` type encapsulates this for us. 

With this change we've passed `BlindAlerter` and `PlayerStore` to `Game` as it is now responsible for alerting and storing results. 

Our `CLI` is now just concerned with:

- Constructing `Game` with its existing dependencies (which we'll refactor next)
- Interpreting user input as method invocations for `Game`

We want to try to avoid doing "big" refactors which leave us in a state of failing tests for extended periods as that increases the chances of mistakes.

The first thing we'll do is refactor `Game` so that we inject it into `CLI`. We'll do the smallest changes in our tests to facilitate that and then we'll see how we can break up the tests into the themes of parsing user input and game management.

All we need to do right now is change `NewCLI`

```go
func NewCLI(in io.Reader, out io.Writer, game *Game) *CLI {
	return &CLI{
		in:  bufio.NewReader(in),
		out: out,
		game: game,
	}
}
```

This feels like an improvement already. We have less dependencies and our dependency list is reflecting our overall design goal of CLI being concerned with input/output and delegating game specific actions to a `Game`.

If you try and compile there are problems. You should be able to fix these problems yourself. Don't worry about making any mocks for `Game` right now, just initialise _real_ `Game`s just to get everything compiling and tests green.

To do this you'll need to make a constructor

```go
func NewGame(alerter BlindAlerter, store PlayerStore) *Game {
	return &Game{
		alerter:alerter,
		store:store,
	}
}
```

Here's an example of one of the setups for the tests being fixed

```go
stdout := &bytes.Buffer{}
in := strings.NewReader("7\n")
blindAlerter := &SpyBlindAlerter{}
game := poker.NewGame(blindAlerter, dummyPlayerStore)

cli := poker.NewCLI(in, stdout, game)
cli.PlayPoker()
```

It shouldn't take much effort to fix the tests and be back to green again (that's the point!) but make sure you fix `main.go` too before the next stage.

```go
// main.go
game := poker.NewGame(poker.BlindAlerterFunc(poker.StdOutAlerter), store)
cli := poker.NewCLI(os.Stdin, os.Stdout, game)
cli.PlayPoker()
```
