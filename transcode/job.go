package transcode

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"path"
// 	"regexp"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"github.com/rs/zerolog/log"

// 	"github.com/voc/stream-api/client"
// 	"github.com/voc/stream-api/stream"
// 	"github.com/voc/stream-api/systemd"
// 	"go.etcd.io/etcd/clientv3"
// )

// var unitRegexp = regexp.MustCompile(`^transcode@(.+)\.target$`)
// var unitFormat = "transcode@%s.target"

// type jobConfig struct {
// 	stream     *stream.Stream
// 	lease      clientv3.LeaseID
// 	api        client.TranscoderAPI
// 	configPath string
// 	sinks      []string
// }

// // transcoderJob represents a single running stream transcode
// type transcoderJob struct {
// 	conf *jobConfig
// 	conn *systemd.Conn

// 	done    sync.WaitGroup
// 	stopped atomic.Value
// 	cancel  context.CancelFunc
// }

// // newJob creates a new transcoding job
// func newJob(parentContext context.Context, conf *jobConfig) (*transcoderJob, error) {
// 	conn, err := systemd.Connect()
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx, cancel := context.WithCancel(parentContext)
// 	j := &transcoderJob{
// 		conn:   conn,
// 		conf:   conf,
// 		cancel: cancel,
// 	}
// 	j.stopped.Store(false)
// 	j.done.Add(1)
// 	go j.run(ctx)
// 	return j, nil
// }

// // run starts the jobs maintenance loop
// func (j *transcoderJob) run(ctx context.Context) {
// 	defer j.done.Done()
// 	defer j.stopped.Store(true)
// 	defer j.cancel()
// 	ticker := time.NewTicker(transcoderTTL / 2)
// 	defer ticker.Stop()

// 	// template config and restart on config change
// 	if j.templateConfig() {
// 		j.restartUnit(ctx)
// 	} else {
// 		j.startUnit(ctx)
// 	}
// 	j.enableUnit(ctx)
// 	defer j.stopUnit(context.Background())
// 	defer j.removeConfig()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			revokeCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
// 			defer cancel()
// 			err := j.conf.api.RevokeLease(revokeCtx, j.conf.lease)
// 			if err != nil {
// 				log.Error().Err(err).Msg("transcoder/job/lease")
// 			}
// 			return
// 		case <-ticker.C:
// 			err := j.conf.api.RefreshLease(ctx, j.conf.lease)
// 			if err != nil {
// 				log.Error().Err(err).Msg("transcoder/job/lease")
// 				return
// 			}
// 			// j.syncUnitState(ctx)
// 		}
// 	}
// }

// func (j *transcoderJob) configFile() string {
// 	return path.Join(j.conf.configPath, j.conf.stream.Slug)
// }

// // templateConfig templates the transcoder config
// // if the file on disk changed it returns true
// func (j *transcoderJob) templateConfig() bool {
// 	var buf bytes.Buffer
// 	type StreamConfig struct {
// 		Slug       string
// 		Format     string
// 		OutputType string
// 		Source     string
// 		Sinks      []string
// 	}
// 	err := configTemplate.Execute(&buf, &StreamConfig{
// 		Slug:   j.conf.stream.Slug,
// 		Format: j.conf.stream.Format,
// 		Source: j.conf.stream.Source,
// 		Sinks:  j.conf.sinks,
// 	})
// 	if err != nil {
// 		log.Fatal().Err(err).Msg("transcoder/job: templateConfig")
// 	}
// 	newConf := buf.Bytes()

// 	filename := j.configFile()
// 	oldConf, err := ioutil.ReadFile(filename)
// 	fmt.Println("got file", filename, string(oldConf))
// 	if err == nil && bytes.Compare(oldConf, newConf) == 0 {
// 		return false
// 	}
// 	err = os.WriteFile(filename, newConf, 0644)
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: writeConfig")
// 	}
// 	fmt.Println("wrote file", string(filename), string(newConf))
// 	return true
// }

// // removeConfig cleans up the config file
// func (j *transcoderJob) removeConfig() {
// 	filename := j.configFile()
// 	if err := os.Remove(filename); err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: config remove")
// 	}
// }

// // unitName returns the systemd unit name for this transcoding job
// func (j *transcoderJob) unitName() string {
// 	return fmt.Sprintf(unitFormat, j.conf.stream.Slug)
// }

// func (j *transcoderJob) startUnit(ctx context.Context) {
// 	log.Debug().Msgf("transcoder/job: start %s", j.unitName())
// 	err := j.conn.StartUnit(ctx, j.unitName())
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: startUnit")
// 	}
// }

// func (j *transcoderJob) restartUnit(ctx context.Context) {
// 	log.Debug().Msgf("transcoder/job: restart %s", j.unitName())
// 	err := j.conn.RestartUnit(ctx, j.unitName())
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: restartUnit")
// 	}
// }

// func (j *transcoderJob) enableUnit(ctx context.Context) {
// 	log.Debug().Msgf("transcoder/job: enable %s", j.unitName())
// 	err := j.conn.EnableUnit(ctx, j.unitName())
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: enableUnit")
// 	}
// }

// func (j *transcoderJob) stopUnit(ctx context.Context) {
// 	err := j.conn.DisableUnit(ctx, j.unitName())
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: disableUnit")
// 	}
// 	err = j.conn.StopUnit(ctx, j.unitName())
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/job: stopUnit")
// 	}
// }

// // syncState syncs transcoding jobs with running systemd units
// // not needed if unit can't fail (Restart=always, StartLimitInterval=0)
// func (j *transcoderJob) syncUnitState(ctx context.Context) {
// 	units, err := j.conn.ListUnits(ctx)
// 	if err != nil {
// 		log.Error().Err(err).Msg("transcoder/sync: listUnits")
// 	}

// 	// reenable failed unit
// 	for _, unit := range units {
// 		res := unitRegexp.FindSubmatch([]byte(unit.Name))
// 		if res == nil || string(res[1]) != j.conf.stream.Slug {
// 			continue
// 		}

// 		fmt.Println(unit, j.conf.stream.Slug)

// 		if unit.ActiveState == "failed" {
// 			log.Info().Msgf("transcoder/sync: restarting "+unitFormat, j.conf.stream.Slug)
// 			j.restartUnit(ctx)
// 		}
// 	}
// }

// func (j *transcoderJob) Stop() {
// 	j.cancel()
// }

// func (j *transcoderJob) Wait() {
// 	j.done.Wait()
// }

// func (j *transcoderJob) Stopped() bool {
// 	return j.stopped.Load().(bool)
// }
