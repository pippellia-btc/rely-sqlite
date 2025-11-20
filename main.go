package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/pippellia-btc/rely"
	sqlite "github.com/vertex-lab/nostr-sqlite"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := sqlite.New("store.sqlite")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	relay := rely.NewRelay(
		rely.WithQueueCapacity(10_000),
		rely.WithMaxProcessors(10),
	)

	relay.On.Event = Save(db)
	relay.On.Req = Query(db)
	relay.On.Count = Count(db)

	err = relay.StartAndServe(ctx, "localhost:3334")
	if err != nil {
		panic(err)
	}
}

func Save(db *sqlite.Store) func(_ rely.Client, e *nostr.Event) error {
	return func(_ rely.Client, e *nostr.Event) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		switch {
		case nostr.IsRegularKind(e.Kind):
			_, err := db.Save(ctx, e)
			return err

		case nostr.IsReplaceableKind(e.Kind) || nostr.IsAddressableKind(e.Kind):
			_, err := db.Replace(ctx, e)
			return err

		default:
			// the event will only be broadcasted, but not stored
			return nil
		}
	}
}

func Query(db *sqlite.Store) func(ctx context.Context, _ rely.Client, filters nostr.Filters) ([]nostr.Event, error) {
	return func(ctx context.Context, _ rely.Client, filters nostr.Filters) ([]nostr.Event, error) {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		return db.Query(ctx, filters...)
	}
}

func Count(db *sqlite.Store) func(_ rely.Client, filters nostr.Filters) (count int64, approx bool, err error) {
	return func(_ rely.Client, filters nostr.Filters) (count int64, approx bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		count, err = db.Count(ctx, filters...)
		if err != nil {
			return -1, false, err
		}
		return count, false, nil
	}
}
