package services

import (
	"bytes"
	"context"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
)

const usersPerRequest = 5000

type importBatch struct {
	node  sharding.Node
	body  bytes.Buffer
	users int
}

type batcher struct {
	wallets WalletClient
	version int64
	batches map[string]*importBatch
}

func newBatcher(wallets WalletClient, version int64) *batcher {
	return &batcher{wallets: wallets, version: version, batches: map[string]*importBatch{}}
}

func (b *batcher) add(ctx context.Context, node sharding.Node, line []byte) error {
	batch, ok := b.batches[node.ID]
	if !ok {
		batch = &importBatch{node: node}
		b.batches[node.ID] = batch
	}

	batch.body.Write(line)
	batch.body.WriteByte('\n')
	batch.users++

	if batch.users < usersPerRequest {
		return nil
	}

	return b.flush(ctx, batch)
}

func (b *batcher) flushAll(ctx context.Context) error {
	for _, batch := range b.batches {
		if err := b.flush(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (b *batcher) flush(ctx context.Context, batch *importBatch) error {
	if batch.users == 0 {
		return nil
	}

	if err := b.wallets.Import(ctx, batch.node, b.version, batch.body.Bytes()); err != nil {
		return err
	}

	batch.body.Reset()
	batch.users = 0

	return nil
}
