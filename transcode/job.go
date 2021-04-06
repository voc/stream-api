package transcode

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/voc/stream-api/client"
	"github.com/voc/stream-api/stream"
	"github.com/voc/stream-api/systemd"
	"go.etcd.io/etcd/clientv3"
)

type transcoderJob struct {
	stream     *stream.Stream
	lease      clientv3.LeaseID
	api        client.TranscoderAPI
	conn       *systemd.Conn
	configPath string

	done    sync.WaitGroup
	stopped atomic.Value
	cancel  context.CancelFunc
}

func newJob(parentContext context.Context, stream *stream.Stream, lease clientv3.LeaseID, api client.TranscoderAPI, configPath string) (*transcoderJob, error) {
	conn, err := systemd.Connect()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentContext)
	job := &transcoderJob{
		stream:     stream,
		lease:      lease,
		api:        api,
		conn:       conn,
		configPath: configPath,

		cancel: cancel,
	}
	job.stopped.Store(false)
	job.done.Add(1)
	go job.run(ctx)
	return job, nil
}

func (j *transcoderJob) run(ctx context.Context) {
	defer j.done.Done()
	defer j.stopped.Store(true)
	defer j.cancel()
	ticker := time.NewTicker(transcoderTTL / 2)
	defer ticker.Stop()

	// template config
	if j.templateConfig() {
		j.startUnit(ctx)
	}
	defer j.stopUnit(context.Background())

	for {
		select {
		case <-ctx.Done():
			revokeCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			err := j.api.RevokeLease(revokeCtx, j.lease)
			if err != nil {
				log.Error().Err(err).Msg("transcoder/job/lease")
			}
			return
		case <-ticker.C:
			err := j.api.RefreshLease(ctx, j.lease)
			if err != nil {
				log.Error().Err(err).Msg("transcoder/job/lease")
				return
			}
		}
	}
}

type StreamConfig struct {
	Slug       string
	Format     string
	OutputType string
	Source     string
	Sink       string
}

var configTemplate = template.Must(template.New("transcoderConfig").Parse(`
stream_key={{ .Slug }}
format={{ .Format }}
output={{ .OutputType }}
transcoding_source={{ .Source }}
transcoding_sink={{ .Sink }}
`))

func (j *transcoderJob) templateConfig() bool {
	var buf bytes.Buffer
	err := configTemplate.Execute(&buf, &StreamConfig{
		Slug:   j.stream.Slug,
		Format: j.stream.Format,
		Source: j.stream.Source,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("transcoder/job: templateConfig")
	}
	newConf := buf.Bytes()

	filename := j.configPath + "/" + j.stream.Slug
	oldConf, err := ioutil.ReadFile(filename)
	if err == nil && bytes.Compare(oldConf, newConf) == 0 {
		return false
	}
	err = os.WriteFile(filename, newConf, 0644)
	if err != nil {
		log.Error().Err(err).Msg("transcoder/job: writeConfig")
	}
	return true
}

func (j *transcoderJob) removeConfig() {
	filename := j.configPath + "/" + j.stream.Slug
	if err := os.Remove(filename); err != nil {
		log.Error().Err(err).Msg("transcoder/job: config remove")
	}
}

func (j *transcoderJob) startUnit(ctx context.Context) {
	unitName := fmt.Sprintf("transcode@%s", j.stream.Slug)
	err := j.conn.RestartUnit(ctx, unitName)
	if err != nil {
		log.Error().Err(err).Msg("transcoder/job: restartUnit")
	}
	err = j.conn.EnableUnit(ctx, unitName)
	if err != nil {
		log.Error().Err(err).Msg("transcoder/job: enableUnit")
	}
}

func (j *transcoderJob) stopUnit(ctx context.Context) {
	unitName := fmt.Sprintf("transcode@%s", j.stream.Slug)
	err := j.conn.DisableUnit(ctx, unitName)
	if err != nil {
		log.Error().Err(err).Msg("transcoder/job: disableUnit")
	}
	err = j.conn.StopUnit(ctx, unitName)
	if err != nil {
		log.Error().Err(err).Msg("transcoder/job: stopUnit")
	}
}

func (j *transcoderJob) Stop() {
	j.cancel()
}

func (j *transcoderJob) Wait() {
	j.done.Wait()
}

func (j *transcoderJob) Stopped() bool {
	return j.stopped.Load().(bool)
}
