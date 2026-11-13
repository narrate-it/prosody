package seslink

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

const maxEnvelopeBytes = 256 << 10

// ExtractJobID routes an SNS Notification to its canonical AMOS job. It is a
// routing projection only: authentication is exclusively the verified source's
// authority, and nothing here treats a message as trusted evidence.
func ExtractJobID(raw []byte) (uuid.UUID, bool) {
	if len(raw) == 0 || len(raw) > maxEnvelopeBytes {
		return uuid.UUID{}, false
	}
	var envelope struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Type != "Notification" {
		return uuid.UUID{}, false
	}
	var event struct {
		EventType string `json:"eventType"`
		Mail      struct {
			Tags map[string][]string `json:"tags"`
		} `json:"mail"`
	}
	if json.Unmarshal([]byte(envelope.Message), &event) != nil || event.EventType != "Send" {
		return uuid.UUID{}, false
	}
	values := event.Mail.Tags["amos-job-id"]
	if len(values) != 1 {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(values[0])
	if err != nil || id.Version() != 7 || id.Variant() != uuid.RFC4122 || id.String() != values[0] {
		return uuid.UUID{}, false
	}
	return id, true
}

// RunIngest consumes the subscription queue until ctx ends. Every envelope is
// authenticated before protected retention; unauthenticated, foreign or
// non-Send messages are discarded without logging their contents. Retention
// failures leave the message for redelivery. Envelopes may contain recipients,
// headers and private content: none of it is ever logged or exposed.
func (s *Subsystem) RunIngest(ctx context.Context) {
	for ctx.Err() == nil {
		out, err := s.queue.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.custody.QueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   60,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("seslink: ingest receive unavailable: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		for _, m := range out.Messages {
			if ctx.Err() != nil {
				return
			}
			s.process(ctx, m)
		}
	}
}

func (s *Subsystem) process(ctx context.Context, m sqstypes.Message) {
	raw := []byte(aws.ToString(m.Body))
	jobID, ok := ExtractJobID(raw)
	if !ok {
		s.delete(ctx, m)
		return
	}
	if _, err := s.Verified.Verify(ctx, jobID, raw); err != nil {
		if ctx.Err() != nil {
			return
		}
		// Unauthenticated evidence is never retained as trusted; discard.
		s.delete(ctx, m)
		return
	}
	if err := s.Store.Retain(ctx, jobID, raw); err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("seslink: envelope retention unavailable; leaving message for redelivery")
		return
	}
	s.delete(ctx, m)
}

func (s *Subsystem) delete(ctx context.Context, m sqstypes.Message) {
	_, err := s.queue.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.custody.QueueURL),
		ReceiptHandle: m.ReceiptHandle,
	})
	if err != nil && ctx.Err() == nil {
		log.Printf("seslink: ingest delete unavailable: %v", err)
	}
}
