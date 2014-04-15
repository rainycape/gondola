package httpclient_test

import (
	"fmt"
	"net/http"

	"gnd.la/net/httpclient"
)

func ExampleIter() {
	// Passing nil only works on non-App Engine and while
	// running tests. Usually you should pass a *app.Context
	// to httpclient.New.
	c := httpclient.New(nil)
	req, err := http.NewRequest("GET", "http://httpbin.org/redirect/3", nil)
	if err != nil {
		panic(err)
	}
	iter := c.Iter(req)
	// Don't forget to close the Iter after you're done with it
	defer iter.Close()
	var urls []string
	for iter.Next() {
		urls = append(urls, iter.Response().URL().String())
	}
	// iter.Assert() could also be used here
	if err := iter.Err(); err != nil {
		panic(err)
	}
	fmt.Println("Last", iter.Response().URL())
	fmt.Println("Intermediate", urls)
	// Output:
	// Last http://httpbin.org/get
	// Intermediate [http://httpbin.org/redirect/3 http://httpbin.org/redirect/2 http://httpbin.org/redirect/1]
}
