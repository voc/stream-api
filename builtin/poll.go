package builtin

import (
	"time"
    "context"
)

type PollArgs struct {
	Interval time.Duration
	Url string
}

type Poll struct {
	ps PubSub
}

func NewPoll(ctx context.Context, ps PubSub, args PollArgs) *Poll {
	p := &Poll{}

    ticker := time.NewTicker(args.Interval)
    go func() {
        for {
            select {
            case <-ctx.Done():
            	ticker.Stop()
                return
            case t := <-ticker.C:
                p.poll()
            }
        }
    }()

    return p
}

// poll requests and parses data from the icecast server
func (poll *Poll) poll() {
	url := poll.Url
	data, err := pollRequest()
	streams = pollParse(val)
	ps.Publish("streams", streams)
}

// request performs a GET request
func pollRequest() string, error {

}

// parse parses streams from icecast status response
func pollParse() Streams {

}