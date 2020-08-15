package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/RichardKnop/machinery/v1"
)

type worker struct {
	db    *sqlx.DB
	redis *redis.Client
	smtp  struct {
		client *smtp.Client
		from   string
	}

	webserver string // hostname of the main ci web server

	block *crypto.Block // used for decrypting ssh keys
	log   *log.Logger

	driverconf map[string]map[string]interface{} // global driver config
	drivers    *driver.Registry                  // configured drivers

	timeout    time.Duration

	server     *machinery.Server
	worker     *machinery.Worker

	placer     runner.Placer
	collector  runner.Collector

	builds     *build.Store
}

var (
	subjfmt = "Djinn - Build #%d %s"
	msgfmt  = `Job %s in build #%d %s

Build: %s
Job:   %s

%s`

	failedGrammar = map[runner.Status]string{
		runner.Killed:   "killed",
		runner.Failed:   "failed",
		runner.TimedOut: "timed out",
	}
)

func sendmail(cli *smtp.Client, subject, from string, to []string, msg string) error {
	buf := bytes.NewBufferString("From: " + from + "\r\n")
	buf.WriteString("To: ")

	for i, rcpt := range to {
		buf.WriteString(rcpt)

		if i != len(to) - 1 {
			buf.WriteString("; ")
		}
	}

	buf.WriteString("\r\nSubject: " + subject + "\r\n\r\n")
	buf.WriteString(msg)

	if err := cli.Mail(from); err != nil {
		return errors.Err(err)
	}

	for _, rcpt := range to {
		if err := cli.Rcpt(rcpt); err != nil {
			// handle
		}
	}

	w, err := cli.Data()

	if err != nil {
		return errors.Err(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return errors.Err(err)
	}
	return errors.Err(w.Close())
}

func (w *worker) init(name string, concurrency int) {
	w.server.RegisterTask("run_build", w.run)
	w.worker = w.server.NewWorker("thrall-worker-"+name, concurrency)
	w.builds = build.NewStore(w.db)
}

func (w *worker) qemuRealPath(b *build.Build, disks string) func(string, string) (string, error) {
	return func(arch, name string) (string, error) {
		i, err := image.NewStore(w.db).Get(
			query.Where("user_id", "=", b.UserID),
			query.Where("name", "=", name),
		)

		if err != nil {
			return "", err
		}

		if i.IsZero() {
			name = filepath.Join(strings.Split(name, "/")...)
			return filepath.Join(disks, "_base", arch, name), nil
		}
		return filepath.Join(disks, i.Hash), nil
	}
}

func (w *worker) run(id int64) error {
	b, err := w.builds.Get(query.Where("id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	r := buildRunner{
		db:        w.db,
		build:     b,
		log:       w.log,
		block:     w.block,
		collector: w.collector,
		placer:    w.placer,
		buf:       &bytes.Buffer{},
		bufs:      make(map[int64]*bytes.Buffer),
		jobs:      make(map[string]*build.Job),
	}

	if b.Status == runner.Killed {
		if err := w.builds.Finished(b.ID, "build killed", b.Status); err != nil {
			return errors.Err(err)
		}
		return errors.Err(r.updateJobs())
	}

	buildDriver, err := build.NewDriverStore(w.db, b).Get()

	if err != nil {
		return errors.Err(err)
	}

	cfg := make(map[string]string)
	json.Unmarshal([]byte(buildDriver.Config), &cfg)

	driverInit, err := w.drivers.Get(cfg["type"])

	if err != nil {
		fmt.Fprintf(r.buf, "driver %s has not been configured for the worker\n", cfg["type"])
		fmt.Fprintf(r.buf, "killing build...\n")

		if err := w.builds.Finished(b.ID, r.buf.String(), runner.Killed); err != nil {
			return errors.Err(err)
		}
		return errors.Err(r.updateJobs())
	}

	if err := r.load(); err != nil {
		return errors.Err(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	sub := w.redis.Subscribe(fmt.Sprintf("kill-%v", b.ID))
	defer sub.Close()

	go func() {
		msg := <-sub.Channel()

		if msg == nil {
			return
		}
		if msg.Payload == b.Secret.String {
			cancel()
		}
	}()

	merged := make(map[string]interface{})

	for k, v := range cfg {
		merged[k] = v
	}

	for k, v := range w.driverconf[cfg["type"]] {
		merged[k] = v
	}

	d := driverInit(io.MultiWriter(r.buf, r.driverBuffer()), merged)

	if q, ok := d.(*qemu.QEMU); ok {
		q.Image = strings.Replace(q.Image, "..", "", -1)
		q.Realpath = w.qemuRealPath(b, merged["disks"].(string))
	}

	status, err := r.run(ctx, d)

	if err != nil {
		return errors.Err(err)
	}

	// Send email to the build user, and namespace collaborators if we have
	// SMTP configured for the worker.
	if status == runner.Killed || status == runner.Failed || status == runner.TimedOut {
		if w.smtp.client == nil {
			return nil
		}

		to := make([]string, 0)

		users := user.NewStore(w.db)

		u, err := users.Get(query.Where("id", "=", b.UserID))

		if err != nil {
			return errors.Err(err)
		}

		b.User = u

		to = append(to, u.Email)

		if b.NamespaceID.Valid {
			uu, err := users.All(
				query.WhereQuery("id", "IN",
					namespace.CollaboratorSelect("user_id",
						query.Where("namespace_id", "=", b.NamespaceID),
					),
				),
			)

			if err != nil {
				return errors.Err(err)
			}

			for _, u := range uu {
				to = append(to, u.Email)
			}
		}

		j, err := build.NewJobStore(w.db, b).Get(
			query.Where("status", "IN", runner.Killed, runner.Failed, runner.TimedOut),
			query.OrderDesc("finished_at"),
		)

		if err != nil {
			return errors.Err(err)
		}

		subj := fmt.Sprintf(subjfmt, b.ID, failedGrammar[status])
		msg := fmt.Sprintf(
			msgfmt,
			j.Name,
			b.ID,
			failedGrammar[status],
			w.webserver + b.Endpoint(),
			w.webserver + j.Endpoint(),
			j.Output.String,
		)

		return errors.Err(sendmail(w.smtp.client, subj, w.smtp.from, to, msg))
	}
	return nil
}
