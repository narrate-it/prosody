package main

import (
	"context"

	"github.com/ajent-social/amos/delivery/email/ses"
	"github.com/ajent-social/amos/storage"
	"github.com/google/uuid"
	"github.com/narrate-it/prosody/backend/seslink"
)

// openSES composes the consumer-owned SES verified-event subsystem from
// operator custody (seslink): the SNS topic subscribed to SES Send events, SQS
// ingestion authenticating every envelope before protected PostgreSQL
// retention, and trusted signing-cert pins. These are operator prerequisites;
// the host never fabricates verified delivery events and refuses to start
// without them.
func openSES(ctx context.Context, root string, inst installation, db storage.RuntimeConfig) (*seslink.Subsystem, error) {
	c, err := seslink.LoadCustody(root)
	if err != nil {
		return nil, err
	}
	realm := ses.Realm{
		InstallationID: uuid.MustParse(inst.Scope.InstallationID),
		ApplicationID:  uuid.MustParse(inst.Scope.ApplicationID),
		EnvironmentID:  uuid.MustParse(inst.Scope.EnvironmentID),
	}
	return seslink.Open(ctx, c, realm, db)
}
