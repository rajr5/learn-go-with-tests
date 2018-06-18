package main

import (
	"testing"
)

func TestSearch(t *testing.T) {
	dict := map[string]string{"test": "this is just a test"}

	t.Run("known word", func(t *testing.T) {
		got, _ := Search(dict, "test")
		want := "this is just a test"

		assertStrings(t, got, want)
	})

	t.Run("unknown word", func(t *testing.T) {
		_, got := Search(dict, "unknown")

		assertError(t, got, NotFoundError)
	})
}

func TestAdd(t *testing.T) {
	dict := map[string]string{}
	word := "test"
	def := "this is just a test"

	Add(dict, word, def)

	assertDef(t, dict, word, def)
}

func assertStrings(t *testing.T, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("got '%s' want '%s'", got, want)
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if got != want {
		t.Errorf("got error '%s' want '%s'", got, want)
	}
}

func assertDef(t *testing.T, dict map[string]string, word, def string) {
	t.Helper()

	got, err := Search(dict, word)
	if err != nil {
		t.Fatal("should find added word:", err)
	}

	if def != got {
		t.Errorf("got '%s' want '%s'", got, def)
	}
}
