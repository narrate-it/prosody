// Package seslink composes Prosody's consumer-owned SES verified-event
// subsystem around the public AMOS contract (delivery/email/ses): operator
// custody binding the SNS topic, SES identity and configuration set; trusted
// regional signing-certificate pins; SQS ingestion that authenticates every
// envelope before protected retention; and an insert-only PostgreSQL envelope
// journal implementing ses.RawSNSEventSource. AMOS supplies neither ingestion
// nor a durable journal; this package is that composition. It never fabricates
// verified delivery evidence and never logs envelope contents.
package seslink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ajent-social/amos/delivery/email/ses"
	"github.com/ajent-social/amos/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	// Origin and From bind the sender to the Prosody native account issuer.
	Origin = "https://api.prosody.chat"
	From   = "noreply@prosody.chat"
)

// CertificatePin is one trusted regional SNS signing certificate. The PEM must
// be obtained independently over verified SNS HTTPS, its provenance validated
// and its fingerprint approved before installation. Pins are privileged trust
// configuration; rotation is explicit with bounded overlap (at most two).
type CertificatePin struct {
	SigningCertURL string `json:"signingCertUrl"`
	CertificatePEM string `json:"certificatePem"`
}

// Custody is the operator-provisioned verified-event configuration (ses.json).
// Every field is a prerequisite; absence is a visible startup refusal.
type Custody struct {
	Region              string           `json:"region"`
	AccountID           string           `json:"accountId"`
	TopicARN            string           `json:"topicArn"`
	IdentityARN         string           `json:"identityArn"`
	ConfigurationSet    string           `json:"configurationSet"`
	CredentialProfile   string           `json:"credentialProfile"`
	QueueURL            string           `json:"queueUrl"`
	TrustedCertificates []CertificatePin `json:"trustedCertificates"`
}

// LoadCustody reads and validates the operator SES custody file.
func LoadCustody(root string) (Custody, error) {
	var c Custody
	data, err := os.ReadFile(filepath.Join(root, "ses.json"))
	if err != nil {
		return c, fmt.Errorf("SES verified-event subsystem not provisioned: ses.json custody required (SNS topic, SQS ingestion, envelope retention and signing-cert pins): %w", err)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("SES custody: %w", err)
	}
	if err := c.validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Custody) validate() error {
	if c.Region == "" || c.AccountID == "" || c.TopicARN == "" || c.IdentityARN == "" ||
		c.ConfigurationSet == "" || c.CredentialProfile == "" || c.QueueURL == "" ||
		len(c.TrustedCertificates) == 0 || len(c.TrustedCertificates) > 2 {
		return errors.New("SES custody is incomplete; refusing to start without verified-event prerequisites")
	}
	for _, pin := range c.TrustedCertificates {
		if pin.SigningCertURL == "" || pin.CertificatePEM == "" {
			return errors.New("SES custody pins require both the exact SigningCertURL and the retained certificate PEM")
		}
	}
	return nil
}

// Subsystem owns the admitted sender, the verified source, the protected
// envelope journal and the ingestion queue client.
type Subsystem struct {
	Sender   *ses.Sender
	Verified *ses.SNSVerifiedSource
	Store    *EnvelopeStore
	custody  Custody
	queue    *sqs.Client
}

type profileResolver struct{}

func (profileResolver) ResolveAWS(ctx context.Context, ref string) (aws.CredentialsProvider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithSharedConfigProfile(ref))
	if err != nil {
		return nil, err
	}
	return cfg.Credentials, nil
}

// Open admits the verified-event subsystem. The envelope journal must already
// exist (schema.sql applied with owner custody); Open never installs schema.
func Open(ctx context.Context, c Custody, realm ses.Realm, db storage.RuntimeConfig) (*Subsystem, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	pins := make([]ses.SNSCertificatePin, 0, len(c.TrustedCertificates))
	for _, p := range c.TrustedCertificates {
		pins = append(pins, ses.SNSCertificatePin{SigningCertURL: p.SigningCertURL, CertificatePEM: []byte(p.CertificatePEM)})
	}
	store, err := openEnvelopeStore(ctx, db)
	if err != nil {
		return nil, err
	}
	verified, err := ses.NewSNSVerifiedSource(ses.SNSConfig{
		Region: c.Region, AccountID: c.AccountID, TopicARN: c.TopicARN, IdentityARN: c.IdentityARN,
		ConfigurationSet: c.ConfigurationSet, Realm: realm, TrustedCertificates: pins,
	}, store)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("SES verified source: %w", err)
	}
	sender, err := ses.New(ctx, ses.Config{
		Region: c.Region, CredentialRef: c.CredentialProfile, FromAddress: From,
		ApplicationOrigin: Origin, ConfigurationSet: c.ConfigurationSet, Realm: realm,
	}, profileResolver{}, verified)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("SES sender: %w", err)
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithSharedConfigProfile(c.CredentialProfile), awsconfig.WithRegion(c.Region))
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("SES ingestion credentials: %w", err)
	}
	return &Subsystem{Sender: sender, Verified: verified, Store: store, custody: c, queue: sqs.NewFromConfig(awsCfg)}, nil
}

// Close releases the envelope journal pool.
func (s *Subsystem) Close() {
	if s != nil && s.Store != nil {
		s.Store.Close()
	}
}
