package concurrency

import (
	"testing"
	"time"
)

func fakeIsWebsiteOK(url string) bool {
	if url == "http://blog.gypsydave5.com" {
		return false
	}
	return true
}

func slowIsWebsiteOK(_ string) bool {
	time.Sleep(20 * time.Millisecond)
	return true
}

func TestWebsiteChecker(t *testing.T) {
	websites := []string{
		"http://google.com",
		"http://blog.gypsydave5.com",
		"waat://furhurterwe.geds",
	}

	actualResults := WebsiteChecker(fakeIsWebsiteOK, websites)

	expectedResults := map[string]bool{
		"http://google.com":          true,
		"http://blog.gypsydave5.com": false,
		"waat://furhurterwe.geds":    true,
	}

	want := len(expectedResults)
	got := len(actualResults)
	if want != got {
		t.Fatalf("Wanted %v, got %v", want, got)
	}
	assertSameResults(t, expectedResults, actualResults)
}

func BenchmarkWebsiteChecker(b *testing.B) {
	for i := 0; i < b.N; i++ {
		websites := make([]string, 100)
		for index, _ := range websites {
			websites[index] = "http://google.co.uk"
		}

		WebsiteChecker(slowIsWebsiteOK, websites)
	}
}

func assertSameResults(t *testing.T, expectedResults, actualResults map[string]bool) {
	for expectedKey, expectedValue := range expectedResults {
		actualValue, ok := actualResults[expectedKey]
		if !ok {
			t.Fatalf("actual results did not contain expected key: '%s'", expectedKey)
		}
		if actualValue != expectedValue {
			t.Fatalf("expected value of key '%s' in actual results to be '%v', but it was '%v'", expectedKey, expectedValue, actualValue)
		}
	}

	for actualKey, _ := range actualResults {
		if _, ok := expectedResults[actualKey]; !ok {
			t.Fatalf("found unexpected key in actual results: '%s'", actualKey)
		}
	}
}
