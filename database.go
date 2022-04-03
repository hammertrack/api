package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
	"github.com/hammertrack/tracker/errors"
)

var (
	ErrDBConnTimeout = errors.New("test connection with database timed out")
)

type SubscribedStatus int

func (s SubscribedStatus) MarshalCQL(info gocql.TypeInfo) ([]byte, error) {
	return gocql.Marshal(info, int(s))
}

func (s *SubscribedStatus) UnmarshalCQL(info gocql.TypeInfo, data []byte) error {
	var n int
	if err := gocql.Unmarshal(info, data, &n); err != nil {
		return err
	}
	*s = SubscribedStatus(n)
	return nil
}

const (
	SubscribedStatusFalse SubscribedStatus = iota
	SubscribedStatusTrue
	SubscribedStatusUnknown
)

type Ban struct {
	Channel    string           `json:"ch,omitempty"`
	Username   string           `json:"usr,omitempty"`
	At         time.Time        `json:"ts,omitempty"`
	Recent     []string         `json:"msgs,omitempty"`
	Subscribed SubscribedStatus `json:"sub,omitempty"`
}

type Pagination struct {
	After string `json:"after,omitempty"`
}

type ManyBan struct {
	Data       []Ban
	Pagination *Pagination
}

type Channel string

type Driver interface {
	Channels() ([]Channel, error)
	BansByUser(username string, after Cursor) (*ManyBan, error)
	BansByChannel(username string, after Cursor) (*ManyBan, error)
	Close() error
}

type Storage struct {
	driver Driver
}

func (s *Storage) Channels() ([]Channel, error) {
	return s.driver.Channels()
}

func (s *Storage) BansByUser(username string, cursor Cursor) (*ManyBan, error) {
	return s.driver.BansByUser(username, cursor)
}

func (s *Storage) BansByChannel(username string, after Cursor) (*ManyBan, error) {
	return s.driver.BansByChannel(username, after)
}

func (s *Storage) Shutdown() error {
	return s.driver.Close()
}

func NewStorage(d Driver) *Storage {
	return &Storage{
		driver: d,
	}
}

type CassandraDriver struct {
	s      *gocql.Session
	ctx    context.Context
	cancel context.CancelFunc
}

func (d *CassandraDriver) Channels() ([]Channel, error) {
	iter := d.s.Query(`SELECT user_name FROM tracked_channels WHERE shard_id=1`).
		WithContext(d.ctx).
		Iter()

	var (
		scanner = iter.Scanner()
		all     = make([]Channel, 0, iter.NumRows())
		err     error
		ch      string
	)
	for scanner.Next() {
		if err = scanner.Scan(&ch); err != nil {
			return nil, errors.Wrap(err)
		}
		all = append(all, Channel(ch))
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err)
	}
	return all, nil
}

func (d *CassandraDriver) BansByUser(username string, after Cursor) (*ManyBan, error) {
	iter := d.s.Query(`SELECT channel_name, user_name, at, messages, sub
    FROM hammertrack.mod_messages_by_user_name WHERE user_name=?`, username).
		WithContext(d.ctx).
		PageState(after).
		Iter()

	var (
		scanner = iter.Scanner()
		all     = make([]Ban, 0, iter.NumRows())
		err     error
		b       Ban
	)
	for scanner.Next() {
		if err = scanner.Scan(&b.Channel, &b.Username, &b.At, &b.Recent, &b.Subscribed); err != nil {
			return nil, errors.Wrap(err)
		}
		all = append(all, b)
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err)
	}

	var nextAfter string
	if nextState := iter.PageState(); len(nextState) > 0 {
		nextAfter, err = Cursor(iter.PageState()).Obscure()
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return &ManyBan{
		Data: all,
		Pagination: &Pagination{
			After: nextAfter,
		},
	}, nil
}

func (d *CassandraDriver) BansByChannel(username string, after Cursor) (*ManyBan, error) {
	iter := d.s.Query(`SELECT channel_name, user_name, at, messages, sub
    FROM hammertrack.mod_messages_by_channel_name
    WHERE channel_name=? AND month = ?`, username, time.Now().Month()).
		WithContext(d.ctx).
		PageState(after).
		Iter()

	var (
		scanner = iter.Scanner()
		all     = make([]Ban, 0, iter.NumRows())
		err     error
		b       Ban
	)
	for scanner.Next() {
		if err = scanner.Scan(&b.Channel, &b.Username, &b.At, &b.Recent, &b.Subscribed); err != nil {
			return nil, errors.Wrap(err)
		}
		all = append(all, b)
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err)
	}

	var nextAfter string
	if nextState := iter.PageState(); len(nextState) > 0 {
		nextAfter, err = Cursor(iter.PageState()).Obscure()
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return &ManyBan{
		Data: all,
		Pagination: &Pagination{
			After: nextAfter,
		},
	}, nil
}

func (d *CassandraDriver) Close() error {
	// Cancel queries
	d.cancel()
	// Close all connections
	d.s.Close()
	return nil
}

// pingUntil tries to connect to the database. If the database is not ready it will
// try again until the given context is canceled
func pingUntil(ctx context.Context, c *gocql.ClusterConfig) (s *gocql.Session, err error) {
	timer := time.NewTicker(time.Second)
	for {
		select {
		case <-timer.C:
			if s, err = c.CreateSession(); err == nil {
				var t string
				if err = s.Query("SELECT now() FROM system.local").
					WithContext(ctx).
					Consistency(gocql.One).
					Scan(&t); err == nil {
					return
				}
			} else {
				errors.Wrap(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func src() string {
	return fmt.Sprintf("%s:%s", DBHost, DBPort)
}

func Cassandra() *CassandraDriver {
	cluster := gocql.NewCluster(src())
	cluster.Keyspace = DBKeyspace
	cluster.ProtoVersion = 4
	cluster.Consistency = gocql.Quorum
	cluster.PageSize = DBPageSize

	ctx, cancel := context.WithCancel(context.Background())
	ctxPing, cancelPing := context.WithTimeout(ctx, time.Duration(DBConnTimeoutSeconds)*time.Second)
	defer cancelPing()

	log.Print("testing database connection...")
	s, err := pingUntil(ctxPing, cluster)
	if err != nil {
		errors.WrapFatalWithContext(ErrDBConnTimeout, struct {
			Cause string
		}{err.Error()})
	}
	log.Print("  ✓ database connection")
	return &CassandraDriver{
		s: s, ctx: ctx, cancel: cancel,
	}
}
