package utils

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

type testLayer struct {
	filename string
	jsonfile string
	gzip     bool
	tarsum   string
}

var testLayers = []testLayer{
	testLayer{
		filename: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/layer.tar",
		jsonfile: "testdata/46af0962ab5afeb5ce6740d4d91652e69206fc991fd5328c1a94d364ad00e457/json",
		tarsum:   "tarsum+sha256:e58fcf7418d4390dec8e8fb69d88c06ec07039d651fedd3aa72af9972e7d046b"},
}

func TestTarSums(t *testing.T) {
	for _, layer := range testLayers {
		fh, err := os.Open(layer.filename)
		if err != nil {
			t.Errorf("failed to open %s: %s", layer.filename, err)
			continue
		}
		ts := &TarSum{Reader: fh}
		_, err = io.Copy(ioutil.Discard, ts)
		if err != nil {
			t.Errorf("failed to copy from %s: %s", layer.filename, err)
			continue
		}
		var gotSum string
		if len(layer.jsonfile) > 0 {
			jfh, err := os.Open(layer.jsonfile)
			if err != nil {
				t.Errorf("failed to open %s: %s", layer.jsonfile, err)
				continue
			}
			buf, err := ioutil.ReadAll(jfh)
			if err != nil {
				t.Errorf("failed to readAll %s: %s", layer.jsonfile, err)
				continue
			}
			gotSum = ts.Sum(buf)
		} else {
			gotSum = ts.Sum(nil)
		}

		if layer.tarsum != gotSum {
			t.Errorf("expecting [%s], but got [%s]", layer.tarsum, gotSum)
		}

	}
}
