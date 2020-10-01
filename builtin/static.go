package builtin

import (
	"fmt"
	"sort"

	"bitbucket.fem.tu-ilmenau.de/scm/~ischluff/stream-api/service"
)



// type Service interface {
// 	Name() string
// 	Capacity() int
// 	Active() int
// 	Add(Stream) error
// 	Remove(Stream) error
// 	WatchService() chan -> interface{}
// }

// type Source interface {
// 	WatchStreams() chan -> Stream
// }

type Exec struct {
	capacity int
	active int
}

func NewExec(cfg config.Plugin) *Exec {
	return &Exec{
		capacity: 
	}
}

func (ex *Exec) Name() {
	return "transcode"
}

func (ex)

