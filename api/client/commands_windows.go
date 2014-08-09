package client

import "os"

func checkSigChld(s os.Signal) bool {
	return false
}
