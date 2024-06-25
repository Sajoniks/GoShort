package mq

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/task"
	"github.com/sajoniks/GoShort/internal/trace"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type KafkaWriterWorkerInterface interface {
	AddJsonMessage(m any)
}

type writerNoOp struct{}

func (k writerNoOp) AddJsonMessage(any) {
}

func NewWriterNoOp() KafkaWriterWorkerInterface {
	return &writerNoOp{}
}

type KafkaWriterWorker struct {
	writer *kafka.Writer
	logger *zap.Logger
	pool   *task.Pool
}

// NewKafkaWriterWorker Creates a new KafkaWriterWorker instance
//
// Provided logger is wrapped with namespace, so
// there is no need to pass already wrapped logger
//
// Worker utilizes task.Pool
// asynchronously sending messages to the queue
func NewKafkaWriterWorker(config *config.KafkaWriterConfig, logger *zap.Logger) *KafkaWriterWorker {
	w := &kafka.Writer{
		Addr:  kafka.TCP(config.Brokers...),
		Topic: config.Topic,
	}
	writer := &KafkaWriterWorker{
		writer: w,
		pool:   task.NewPool(8, logger.With(zap.Namespace("kafka_pool"))),
		logger: logger.With(
			zap.String("topic", w.Topic),
			zap.String("addr", w.Addr.String()),
		),
	}
	return writer
}

func (k *KafkaWriterWorker) Shutdown() {
	k.pool.Shutdown()
}

func (k *KafkaWriterWorker) AddJsonMessage(m any) {
	k.pool.AddFunc(func(ctx context.Context) {
		bs, err := json.Marshal(m)
		if err != nil {
			k.logger.Error("error marshaling message",
				zap.Error(trace.WrapError(err)),
			)
		}
		err = k.writer.WriteMessages(ctx, kafka.Message{Value: bs})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				k.logger.Error("write cancelled")
			} else {
				k.logger.Error("error writing message",
					zap.Error(trace.WrapError(err)),
				)
			}
		} else {
			k.logger.Info("sent message")
		}
	})
}
