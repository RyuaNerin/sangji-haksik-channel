package share

import (
	"log"
	"runtime"
)

func init() {
	if runtime.GOOS == "windows" {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
	} else {
		log.SetFlags(log.Llongfile)
	}
}
