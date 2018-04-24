# HTTP Server (WIP)

You have been asked to create a web server where users can track how many games players have won

- `GET /players/{name}` should return a number indicating total number of wins
- `POST /players/{name}/win` should increment the number of wins 

We will follow the TDD approach, getting working software as quickly as we can and then making small iterative improvements until we have the solution. By taking this approach we

- Keep the problem space small at any given time
- Don't go down rabbit holes
- If we ever get stuck/lost doing a revert wouldn't lose loads of work.


## Red, green, refactor

Throughout this book we have emphasised the TDD process of write a test & watch it fail (red), write the _minimal_ amount of code to make it work (green) and then refactor.

This discipline of writing the minimal amount of code is important in terms of the safety TDD gives you. You should be striving to get out of "red" as soon as you can.

Kent Beck describes it as 

> Make the test work quickly, committing whatever sins necessary in process.

You can commit these sins because you will refactor afterwards backed by the safety of the tests.

#### What if you don't do this?

The more changes you make while in red, the more likely you are to add more problems, not covered by tests. 

The idea is to be iteratively writing useful code with small steps, driven by tests so that you don't fall into a rabbit hole for hours.

### Chicken and egg

How can we incrementally build this? We cant `GET` a player without having stored something and it seems hard to know if `POST` has worked without the `GET` endpoint already existing. 

This is where _mocking_ shines. 

- `GET` will need a `PlayerStore` _thing_ to get scores for a player. This should be an interface so when we test we can create a simple stub to test our code without needing to have implemented any actual storage code.
- For `POST` we can _spy_ on its calls to `PlayerStore` to make sure it stores players correctly. Our implementation of saving wont be coupled to retrieval.
- For having some working software quickly we can make a very simple in-memory implementation and then later we can create an implementation backed by whatever storage mechanism we prefer. 

## Write the test first

We can write a test and make it pass by returning a hard-coded value to get us started. Kent Beck refers this as "Faking it". Once we have a working test we can then write more tests to help us remove that constant

By doing this very small step, we can make the important start of getting an overall project structure working correctly without having to worry too much about our application logic.

To create a web server in Go you will typically call [https://golang.org/pkg/net/http/#ListenAndServe](ListenAndServe)

```go
func ListenAndServe(addr string, handler Handler) error
```

This will start a web server listening on a port, creating a goroutine for every request and running it against a [`Handler`](https://golang.org/pkg/net/http/#Handler).

```go
type Handler interface {
        ServeHTTP(ResponseWriter, *Request)
}
```

It's has one function which expects two arguments, the first being where we _write our response_ and the second being the HTTP request that was sent to us.

Let's write a test for a function `PlayerServer` that takes in those two arguments. The request sent in will be to get a player's score, which we expect to be `"20"`.

```go
t.Run("returns Pepper's score", func(t *testing.T) {
    req, _ := http.NewRequest(http.MethodGet, "/players/Pepper", nil)
    res := httptest.NewRecorder()

    PlayerServer(res, req)

    got := res.Body.String()
    want := "20"

    if got != want {
        t.Errorf("got '%s', want '%s'", got, want)
    }
})
```

In order to test our server, we will need a `Request` to send in and we'll want to _spy_ on what our handler writes to the `ResponseWriter`. 

- We use `http.NewRequest` to create a request. The first argument is the request's method and the second is the request's path. The `nil` argument refers to the request's body, which we don't need to set in this case.
- `net/http/httptest` has a spy already made for us called `ResponseRecorder` so we can use that. It has many helpful methods to inspect what has been written as a response.

## Try to run the test

`./server_test.go:13:2: undefined: PlayerServer`

## Write the minimal amount of code for the test to run and check the failing test output

The compiler is here to help, just listen to it.

Define `PlayerServer`

```go
func PlayerServer() {}
```

Try again

```
./server_test.go:13:14: too many arguments in call to PlayerServer
	have (*httptest.ResponseRecorder, *http.Request)
	want ()
```

Add the arguments to our function

```go
import "net/http"

func PlayerServer(w http.ResponseWriter, r *http.Request) {

}
```

The code now compiles and the test fails

```
=== RUN   TestGETPlayers/returns_Pepper's_score
    --- FAIL: TestGETPlayers/returns_Pepper's_score (0.00s)
    	server_test.go:20: got '', want '20'
```

## Write enough code to make it pass

From the DI chapter we touched on HTTP servers with a `Greet` function. We learned that net/http's `ResponseWriter` also implements io `Writer` so we can use `fmt.Fprint` to send strings as HTTP responses

```go
func PlayerServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "20")
}
```

The test should now pass.

## Complete the scaffolding

We want to wire this up into an application. This is important because

- We'll have _actual working software_, we don't want to write tests for the sake of it, it's good to see the code in action.
- As we refactor our code, it's likely we will change the structure of the program. We want to make sure this is reflected in our application too as part of the incremental approach.

Create a new file for our application and put this code in.

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	handler := http.HandlerFunc(PlayerServer)
	if err := http.ListenAndServe(":5000", handler); err != nil {
		log.Fatalf("could not listen on port 5000 %v", err)
	}
}
```

So far all of our application code have been in one file, however this isn't best practice for larger projects where you'll want to separate things into different files. 

To run this, do `go build` which will take all the `.go` files in the directory and build you a program. You can then execute it with `./myprogram`. 

### `http.HandlerFunc`

Earlier we explored that the `Handler` interface is what we need to implement in order to make a server. _Typically_ we do that by creating a `struct` and make it implement the interface. However the use-case for structs is for holding data but _currently_ we have no state, so it doesn't feel right to be creating one.

[HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) lets us avoid this.

> The HandlerFunc type is an adapter to allow the use of ordinary functions as HTTP handlers. If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f. 

```go
type HandlerFunc func(ResponseWriter, *Request)
```

So we use this to wrap our `PlayerServer` function so that it now conforms to `Handler`.


### `http.ListenAndServe(":5000"...`

ListenAndServer takes a port to listen on and a `Handler`. If the port is already being listened to it will return an `error` so we are using an `if` statement to capture that scenario and log the problem to the user.

What we're going to do now is write _another_ test to force us into making a positive change to try and move away from the hard-coded value.

## Write the test first

You may have been thinking

> Surely we need some kind of concept of storage to control which player gets what score. It's weird that the values seem so arbitrary in our tests.

Remember we are just trying to take as small as steps as reasonably possible. By writing the test it may drive us toward our goal in an easier step.

We'll add another subtest to our suite which tries to get the score of a different player, which will break our hard-coded approach.

```go
t.Run("returns Floyd's score", func(t *testing.T) {
    req, _ := http.NewRequest(http.MethodGet, "/players/Floyd", nil)
    res := httptest.NewRecorder()

    PlayerServer(res, req)

    got := res.Body.String()
    want := "10"

    if got != want {
        t.Errorf("got '%s', want '%s'", got, want)
    }
})
```

## Try to run the test
```
=== RUN   TestGETPlayers/returns_Pepper's_score
    --- PASS: TestGETPlayers/returns_Pepper's_score (0.00s)
=== RUN   TestGETPlayers/returns_Floyd's_score
    --- FAIL: TestGETPlayers/returns_Floyd's_score (0.00s)
    	server_test.go:34: got '20', want '10'
```

## Write enough code to make it pass

```go
func PlayerServer(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/players/"):]

	if player == "Pepper" {
		fmt.Fprint(w, "20")
		return
	}

	if player == "Floyd" {
		fmt.Fprint(w, "10")
		return
	}
}
```

By doing this the test has forced us to actually look at the request's URL and make some decision. So whilst in our heads we may have been worrying about player stores and interfaces the next logical step actually seems to be about _routing_.

If we did start with the store code the amount of changes we'd have to do would be very large compared to this. **This is a smaller step towards our final goal and was driven by tests**

We're resisting the temptation to use any routing libraries right now, just the smallest step to get our test passing.

`r.URL.Path` returns the path of the request and then we are using slice syntax to slice it past the final slash after `/players/`. It's not very robust but will do the trick for now.

## Refactor

We can simplify the `PlayerServer` by separating out the score retrieval into a function

```go
// PlayerServer currently returns Hello, world given _any_ request
func PlayerServer(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/player/"):]

	fmt.Fprint(w, GetPlayerScore(player))
}

func GetPlayerScore(name string) string {
	if name == "Pepper" {
		return "20"
	}

	if name == "Floyd" {
		return "10"
	}

	return ""
}
```

And we can DRY up some of the code in the tests by making some helpers

```go
func TestGETPlayers(t *testing.T) {
	t.Run("returns Pepper's score", func(t *testing.T) {
		req := newGetScoreRequest("Pepper")
		res := httptest.NewRecorder()

		PlayerServer(res, req)

		assertResponseBody(t, res.Body.String(), "20")
	})

	t.Run("returns Floyd's score", func(t *testing.T) {
		req := newGetScoreRequest("Floyd")
		res := httptest.NewRecorder()

		PlayerServer(res, req)

		assertResponseBody(t, res.Body.String(), "10")
	})
}

func newGetScoreRequest(name string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/players/%s", name), nil)
	return req
}

func assertResponseBody(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("response body is wrong, got '%s' want '%s'", got, want)
	}
}
```

However we still shouldn't be happy. It doesn't feel right that our server knows the scores. 

Our refactoring has made it pretty clear what to do. 

We moved the score calculation out of the main body of our handler into a function `GetPlayerScore`. This feels like the right place to separate the concerns using interfaces. 

Let's move our function we re-factored to be an interface instead

```go
type PlayerStore interface {
	GetPlayerScore(name string) string
}
```

For our `PlayerServer` to be able to use a `PlayerStore`, it will need a reference to one. Now feels like the right time to change our architecture so that our `PlayerServer` is now a `struct`

```go
type PlayerServer struct {
	store PlayerStore
}
```

Finally, we will now implement the `Handler` interface by adding a method to our new struct and putting in our existing handler code

```go
func (p *PlayerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/player/"):]
	fmt.Fprint(w, p.store.GetPlayerScore(player))
}
```

The only other change is we now call our `store.GetPlayerStore` to get the score, rather than the local function we defined (which we can now delete).

Here is the full code listing of our server

```go
type PlayerStore interface {
	GetPlayerScore(name string) string
}

type PlayerServer struct {
	store PlayerStore
}

func (p *PlayerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/player/"):]
	fmt.Fprint(w, p.store.GetPlayerScore(player))
}
```

#### Fix the issues

This was quite a few changes and we know our tests and application will no longer compile but just relax and let the compiler work through it

`./main.go:9:58: type PlayerServer is not an expression`

We need to change our tests to instead create a new instance of our `PlayerServer` and then call its method `ServeHTTP`.

```go
func TestGETPlayers(t *testing.T) {
	server := &PlayerServer{}
	
	t.Run("returns the Pepper's score", func(t *testing.T) {
		req := newGetScoreRequest("Pepper")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertResponseBody(t, res.Body.String(), "20")
	})

	t.Run("returns Floyd's score", func(t *testing.T) {
		req := newGetScoreRequest("Floyd")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertResponseBody(t, res.Body.String(), "10")
	})
}
```

Notice we're still not worrying about making stores _just yet_, we just want the compiler passing as soon as we can. 

You should be in the habit of prioritising having code that compiles and then code that passes the tests. 

By adding more functionality (like stub stores) we are opening ourselves up to potentially _more_ compilation problems.

Now `main.go` won't compile for the same reason.

```go
func main() {
	server := &PlayerServer{}

	if err := http.ListenAndServe(":5000", server); err != nil {
		log.Fatalf("could not listen on port 5000 %v", err)
	}
}
```

Finally everything is compiling but the tests are failing

```
=== RUN   TestGETPlayers/returns_the_Pepper's_score
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
	panic: runtime error: invalid memory address or nil pointer dereference
```

This is because we have not passed in a `PlayerStore` in our tests. We'll need to make a stub one up. 

```go
type StubPlayerStore struct {
	scores map[string]string
}

func (s *StubPlayerStore) GetPlayerScore(name string) string {
	score := s.scores[name]
	return score
}
```

A `map` is a quick and easy way of making a stub key/value store for our tests. Now let's create one of these stores for our tests and send it into our `PlayerServer`

```go
func TestGETPlayers(t *testing.T) {
	store := StubPlayerStore{
		map[string]string{
			"Pepper": "20",
			"Floyd":  "10",
		},
	}
	server := &PlayerServer{&store}

	t.Run("returns the Pepper's score", func(t *testing.T) {
		req := newGetScoreRequest("Pepper")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertResponseBody(t, res.Body.String(), "20")
	})

	t.Run("returns Floyd's score", func(t *testing.T) {
		req := newGetScoreRequest("Floyd")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertResponseBody(t, res.Body.String(), "10")
	})
}
```

Our tests now pass and are looking better. The _intent_ behind our code is clearer now due to the introduction of the store. We're telling the reader that because we have _this data in a `PlayerStore`_ that when you use it with a `PlayerServer` you should get the following responses.

#### Run the application

Now our tests are passing the last thing we need to do to complete this refactor is to check our application is working. The program should start up but you'll get a horrible response if you try and hit the server at `http://localhost:5000/players/Pepper`. 

The reason for this is that we have not passed in a `PlayerStore`.

We'll need to make an implementation of one, but that's difficult right now as we're not storing any meaningful data so it'll have to be hard-coded for the time being. 

```go
type InMemoryPlayerStore struct{}

func (i *InMemoryPlayerStore) GetPlayerScore(name string) string {
	return "123"
}

func main() {
	server := &PlayerServer{&InMemoryPlayerStore{}}

	if err := http.ListenAndServe(":5000", server); err != nil {
		log.Fatalf("could not listen on port 5000 %v", err)
	}
}
```

If you `go build` again and hit the same URL you should get `"123"`. Not great, but until we store data that's the best we can do. 

We have a few options as to what to do next

- Handle the scenario where the player doesn't exist
- Handle the `POST /players/{name}/win` scenario
- What happens if someone hits a completely wrong URL like `/playerz/{name}` ?
- It didn't feel great that our main application was starting up but not actually working. We had to manually test to see the problem.

Whilst the `POST` scenario gets us closer to the "happy path", I feel it'll be easier to tackle the missing player scenario first as we're in that context already. We'll get to the rest later.

## Write the test first

Add a missing player scenario to our existing suite

```go
t.Run("returns 404 on missing players", func(t *testing.T) {
    req := newGetScoreRequest("Apollo")
    res := httptest.NewRecorder()

    server.ServeHTTP(res, req)

    got := res.Code
    want := http.StatusNotFound

    if got != want {
        t.Errorf("got status %d want %d", got, want)
    }
})
```

## Try to run the test

```
=== RUN   TestGETPlayers/returns_404_on_missing_players
    --- FAIL: TestGETPlayers/returns_404_on_missing_players (0.00s)
    	server_test.go:56: got status 200 want 404
```

## Write enough code to make it pass

```go
func (p *PlayerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/players/"):]
	
	w.WriteHeader(http.StatusNotFound)
	
	fmt.Fprint(w, p.store.GetPlayerScore(player))
}
```

Sometimes I heavily roll my eyes when TDD advocates say "make sure you just write the minimal amount of code to make it pass" as it can feel very pedantic. 

But this scenario illustrates the example well. I have done the bare minimum (knowing it is not correct), which is write a `StatusNotFound` on **all responses** but all our tests are passing! 

**By doing the bare minimum to make the tests pass it can highlight gaps in your tests** In our case we are not asserting that we should be getting a `StatusOK` when players _do_ exist in the store.

Update the other two tests to assert on the status and fix the code. 

Here are the new tests

```go
func TestGETPlayers(t *testing.T) {
	store := StubPlayerStore{
		map[string]string{
			"Pepper": "20",
			"Floyd":  "10",
		},
	}
	server := &PlayerServer{&store}

	t.Run("returns Pepper's score", func(t *testing.T) {
		req := newGetScoreRequest("Pepper")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertStatus(t, res.Code, http.StatusOK)
		assertResponseBody(t, res.Body.String(), "20")
	})

	t.Run("returns Floyd's score", func(t *testing.T) {
		req := newGetScoreRequest("Floyd")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertStatus(t, res.Code, http.StatusOK)
		assertResponseBody(t, res.Body.String(), "10")
	})

	t.Run("returns 404 on missing players", func(t *testing.T) {
		req := newGetScoreRequest("Apollo")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)

		assertStatus(t, res.Code, http.StatusNotFound)
	})
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("did not get correct status, got %d, want %d", got, want)
	}
} 

func newGetScoreRequest(name string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/players/%s", name), nil)
	return req
}

func assertResponseBody(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("response body is wrong, got '%s' want '%s'", got, want)
	}
}
```

We're checking the status in all our tests now so I made a helper `assertStatus` to facilitate that. 

Now our first two tests fail because of the 404 instead of 200 so we can fix `PlayerServer`.

```go
func (p *PlayerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/players/"):]
	
	score := p.store.GetPlayerScore(player)

	if score == "" {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprint(w, score)
}
```

