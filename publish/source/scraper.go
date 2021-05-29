package source

import (
	"context"

	"github.com/voc/stream-api/stream"
)

type Scraper interface {
	Scrape(context.Context) ([]*stream.Stream, error)
}
