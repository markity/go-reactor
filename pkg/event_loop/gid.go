package eventloop

import (
	"github.com/petermattis/goid"
)

// get goroutine id
func getGid() (gid int64) {
	return goid.Get()
}
