# IO and sorting (WIP)

**[You can find all the code for this chapter here](https://github.com/quii/learn-go-with-tests/tree/master/io)**

[In the previous chapter](json.md) we continued iterating on our application by adding a new endpoint `/league`. Along the way we learned about how to deal with JSON, embedding types and routing.

Our product-owner is somewhat perturbed by the software losing the scores when the server was restarted. This is because our implementation of our store is in-memory. She is also not pleased that we didn't interpret the `/league` endpoint should return the players ordered by number of wins!

## The code so far

```go
type PlayerStore interface {
	GetPlayerScore(name string) int
	RecordWin(name string)
	GetLeague() []Player
}

type Player struct {
	Name string
	Wins int
}

type PlayerServer struct {
	store PlayerStore
	http.Handler
}

const jsonContentType = "application/json"

func NewPlayerServer(store PlayerStore) *PlayerServer {
	p := new(PlayerServer)

	p.store = store

	router := http.NewServeMux()
	router.Handle("/league", http.HandlerFunc(p.leagueHandler))
	router.Handle("/players/", http.HandlerFunc(p.playersHandler))

	p.Handler = router

	return p
}

func (p *PlayerServer) leagueHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(p.store.GetLeague())
	w.Header().Set("content-type", jsonContentType)
	w.WriteHeader(http.StatusOK)
}

func (p *PlayerServer) playersHandler(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/players/"):]

	switch r.Method {
	case http.MethodPost:
		p.processWin(w, player)
	case http.MethodGet:
		p.showScore(w, player)
	}
}

func (p *PlayerServer) showScore(w http.ResponseWriter, player string) {
	score := p.store.GetPlayerScore(player)

	if score == 0 {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprint(w, score)
}

func (p *PlayerServer) processWin(w http.ResponseWriter, player string) {
	p.store.RecordWin(player)
	w.WriteHeader(http.StatusAccepted)
}
```

```go
func NewInMemoryPlayerStore() *InMemoryPlayerStore {
	return &InMemoryPlayerStore{map[string]int{}}
}

type InMemoryPlayerStore struct {
	store map[string]int
}

func (i *InMemoryPlayerStore) GetLeague() []Player {
	var league []Player
	for name, wins := range i.store {
		league = append(league, Player{name, wins})
	}
	return league
}

func (i *InMemoryPlayerStore) RecordWin(name string) {
	i.store[name]++
}

func (i *InMemoryPlayerStore) GetPlayerScore(name string) int {
	return i.store[name]
}
```

```go
func main() {
	server := NewPlayerServer(NewInMemoryPlayerStore())

	if err := http.ListenAndServe(":5000", server); err != nil {
		log.Fatalf("could not listen on port 5000 %v", err)
	}
}
```

## Store the data

There are dozens of databases we could use for this but we're going to go for a very simple approach. We're going to store the data for this application in a file as JSON. 

This keeps the data very portable and is relatively simple to implement. 

It wont scale especially well but given this is a prototype it'll be fine for now. If our circumstances change and it's no longer appropriate it'll be simple to swap it out for something different because of the `PlayerStore` abstraction we have used.

We will keep the `InMemoryPlayerStore` for now so that the integration tests keep passing as we develop our new store. Once we are confident our new implementation is sufficient to make the integration test pass we will swap it in and then delete `InMemoryPlayerStore`
 
## Write the test first

By now you should be familiar with the interfaces around the standard library for reading data (`io.Reader`), writing data (`io.Writer`) and how we can use the standard library to test these functions without having to use real files.

For this work to be complete we'll need to implement `PlayerStore` so we'll write tests for our store calling the methods we need to implement. We'll start with `GetLeague`.

```go
func TestFileSystemStore(t *testing.T) {

	t.Run("/league from a reader", func(t *testing.T) {
		database := strings.NewReader(`[
			{"Name": "Cleo", "Wins": 10},
			{"Name": "Chris", "Wins": 33}]`)

		store := FileSystemStore{database}

		got := store.GetLeague()

		want := []Player{
			{"Cleo", 10},
			{"Chris", 33},
		}

		assertLeague(t, got, want)
	})
}
```

We're using `strings.NewReader` which will return us a `Reader`, which is what our `FileSystemStore` will use to read data. In `main` we will open a file, which is also a `Reader`.

## Try to run the test

```
# github.com/quii/learn-go-with-tests/json-and-io/v7
./FileSystemStore_test.go:15:12: undefined: FileSystemStore
```

## Write the minimal amount of code for the test to run and check the failing test output

Let's define `FileSystemStore` in a new file

```go
type FileSystemStore struct {
	
}
```

Try again

```
# github.com/quii/learn-go-with-tests/json-and-io/v7
./FileSystemStore_test.go:15:28: too many values in struct initializer
./FileSystemStore_test.go:17:15: store.GetLeague undefined (type FileSystemStore has no field or method GetLeague)
```

It's complaining because we're passing in a `Reader` but not expecting one and it doesnt have `GetLeague` defined yet.

```go
type FileSystemStore struct {
	database io.Reader
}

func (f *FileSystemStore) GetLeague() []Player {
	return nil
}
```

One more try...

```
=== RUN   TestFileSystemStore//league_from_a_reader
    --- FAIL: TestFileSystemStore//league_from_a_reader (0.00s)
    	FileSystemStore_test.go:24: got [] want [{Cleo 10} {Chris 33}]
```

## Write enough code to make it pass

We've read JSON from a reader before

```go
func (f *FileSystemStore) GetLeague() []Player {
	var league []Player
	json.NewDecoder(f.database).Decode(&league)
	return league
}
```

The test should pass. 

## Refactor

We _have_ done this before! Our test code for the server had to decode the JSON from the response. 

Let's try DRYing this up into a function. 

Create a new file called `league.go` and put this inside.

```go
func NewLeague(rdr io.Reader) ([]Player, error) {
	var league []Player
	err := json.NewDecoder(rdr).Decode(&league)
	return league, err
}
```

Call this in our implementation and in our test helper `getLeagueFromResponse` in `server_test.go`

```go
func (f *FileSystemStore) GetLeague() []Player {
	league, _ := NewLeague(f.database)
	return league
}
```

We haven't got a strategy yet for dealing with parsing errors but let's press on.

### Seeking problems

There is a flaw in our implementation. First of all let's remind ourselves how `io.Reader` is defined

```go
type Reader interface {
        Read(p []byte) (n int, err error)
}
```

With our file you can imagine it reading through byte by byte until the end. What happens if you try and `Read` a second time?

Add the following to the end of our current test

```go
// read again
got = store.GetLeague()
assertLeague(t, got, want)
```

We want this to pass, but if you run the test it doesnt. 

The problem is our `Reader` has reached to the end so there is nothing more to read. We need a way to tell it to go back to the start.

[ReadSeeker](https://golang.org/pkg/io/#ReadSeeker) is another interface in the standard library that can help. 

```go
type ReadSeeker interface {
        Reader
        Seeker
}
```

Remember embedding? This is an interface comprised of `Reader` and [`Seeker`](https://golang.org/pkg/io/#Seeker)

```go
type Seeker interface {
        Seek(offset int64, whence int) (int64, error)
}
```

This sounds good, can we change `FileSystemStore` to take this interface instead?

```go
type FileSystemStore struct {
	database io.ReadSeeker
}

func (f *FileSystemStore) GetLeague() []Player {
	f.database.Seek(0, 0)
	league, _ := NewLeague(f.database)
	return league
}
```

Try running the test, it now passes! Happily for us `string.NewReader` that we used in our test also implements `ReadSeeker` so we didn't have to make any other changes.

Next we'll implement `GetPlayerScore`

## Wrapping up

What we've covered:

- The `Seeker` interface and its relation with `Reader` and `Writer`
- Working with files
