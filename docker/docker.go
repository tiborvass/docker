package main

import (
	"io"
	"log"
	"os"
	"net/http"
	"net/url"
)


// Use this key to encode an RPC call into an URL,
// eg. domain.tld/path/to/method?q=get_user&q=gordon
const ARG_URL_KEY = "q"

func CallToURL(host string, cmd string, args []string) *url.URL {
    qValues := make(url.Values)
    for _, v := range args {
        qValues.Add(ARG_URL_KEY, v)
    }
    return &url.URL{
	Scheme:     "http",
	Host:       host,
        Path:       "/" + cmd,
        RawQuery:   qValues.Encode(),
    }
}


func main() {
	var cmd string
	var args []string
	if len(os.Args) >= 2 {
		cmd = os.Args[1]
	}
	if len(os.Args) >= 3 {
		args = os.Args[2:]
	}
	u := CallToURL(os.Getenv("DOCKER"), cmd, args)
	resp, err := http.Get(u.String())
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, resp.Body)
}
