package mq

import (
	"context"
	"errors"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/task"
	"github.com/sajoniks/GoShort/internal/trace"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"sync"
	"time"
)

type KafkaReaderWorker struct {
	reader *kafka.Reader
	logger *zap.Logger
	pool   *task.Pool
	C      <-chan []byte
	cancel context.CancelFunc
	ctx    context.Context
	wg     sync.WaitGroup
}

// NewKafkaReaderWorker creates a new instance of KafkaReaderWorker that asynchronously reads messages
// from Kafka. This worker uses "at most once" message processing - this means that message is auto-commited on receive.
//
// # Provided logger is wrapped with namespace, so there is no need to provide already wrapped logger
//
// KafkaReaderWorker spawns a goroutine that reads messages from the Kafka.
// When message is received without errors, it is sent to KafkaReaderWorker.C channel
//
// Sending is asynchronous and uses task.Pool, so there can be simultaneous message processing.
func NewKafkaReaderWorker(config *config.KafkaReaderConfig, logger *zap.Logger) *KafkaReaderWorker {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  config.Brokers,
		GroupID:  config.GroupId,
		Topic:    config.Topic,
		MaxBytes: config.MaxBytes,
	})
	ch := make(chan []byte)
	ctx, cancel := context.WithCancel(context.Background())
	reader := &KafkaReaderWorker{
		reader: r,
		pool:   task.NewPool(4, logger.With(zap.Namespace("kafka_pool"))),
		logger: logger.With(
			zap.String("topic", r.Config().Topic),
			zap.Strings("addr", r.Config().Brokers),
			zap.String("consumer-group", r.Config().GroupID),
		),
		C:      ch,
		cancel: cancel,
		ctx:    ctx,
	}

	reader.wg.Add(1)
	go func() {
		defer reader.wg.Done()
		for {
			m, err := r.ReadMessage(reader.ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					logger.Info("shutting down reader")
					return
				} else {
					logger.Error("error reading message", zap.Error(trace.WrapError(err)))
					time.Sleep(time.Second)
					continue
				}
			}

			logger.Info("receive message",
				zap.Int("partition", m.Partition),
				zap.Int64("offset", m.Offset),
				zap.ByteString("key", m.Key),
				zap.ByteString("value", m.Value),
			)

			reader.pool.AddFunc(func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
					ch <- m.Value
				}
			})
		}
	}()

	return reader
}

func (r *KafkaReaderWorker) Shutdown() {
	r.cancel()
	r.wg.Wait()
	r.pool.Shutdown()
}
