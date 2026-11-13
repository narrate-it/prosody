package seslink

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func sendEnvelope(t *testing.T, jobID string) []byte {
	t.Helper()
	event := map[string]any{
		"eventType": "Send",
		"mail": map[string]any{
			"tags": map[string][]string{"amos-job-id": {jobID}},
		},
	}
	message, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(map[string]string{"Type": "Notification", "Message": string(message)})
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func TestExtractJobIDAcceptsCanonicalSend(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	got, ok := ExtractJobID(sendEnvelope(t, id.String()))
	if !ok || got != id {
		t.Fatalf("expected canonical job %v, got %v ok=%v", id, got, ok)
	}
}

func TestExtractJobIDRejectsForeignShapes(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	cases := map[string][]byte{
		"empty":            {},
		"oversized":        []byte(strings.Repeat("x", maxEnvelopeBytes+1)),
		"not json":         []byte("{"),
		"confirmation":     []byte(`{"Type":"SubscriptionConfirmation","Message":"{}"}`),
		"non send":         []byte(`{"Type":"Notification","Message":"{\"eventType\":\"Delivery\",\"mail\":{\"tags\":{\"amos-job-id\":[\"` + id.String() + `\"]}}}"}`),
		"missing tags":     []byte(`{"Type":"Notification","Message":"{\"eventType\":\"Send\",\"mail\":{}}"}`),
		"multi tag values": []byte(`{"Type":"Notification","Message":"{\"eventType\":\"Send\",\"mail\":{\"tags\":{\"amos-job-id\":[\"` + id.String() + `\",\"` + id.String() + `\"]}}}"}`),
		"uuid v4":          []byte(`{"Type":"Notification","Message":"{\"eventType\":\"Send\",\"mail\":{\"tags\":{\"amos-job-id\":[\"` + uuid.New().String() + `\"]}}}"}`),
		"noncanonical uuid": []byte(`{"Type":"Notification","Message":"{\"eventType\":\"Send\",\"mail\":{\"tags\":{\"amos-job-id\":[\"` + strings.ToUpper(id.String()) + `\"]}}}"}`),
	}
	for name, raw := range cases {
		if _, ok := ExtractJobID(raw); ok {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestCustodyValidation(t *testing.T) {
	valid := Custody{
		Region: "us-west-2", AccountID: "394824061124",
		TopicARN: "arn:aws:sns:us-west-2:394824061124:prosody-ses-events",
		IdentityARN: "arn:aws:ses:us-west-2:394824061124:identity/prosody.chat",
		ConfigurationSet: "prosody-identity", CredentialProfile: "prosody-ses",
		QueueURL: "https://sqs.us-west-2.amazonaws.com/394824061124/prosody-ses-events",
		TrustedCertificates: []CertificatePin{{SigningCertURL: "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-abc.pem", CertificatePEM: "pem"}},
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("complete custody rejected: %v", err)
	}
	incomplete := valid
	incomplete.QueueURL = ""
	if err := incomplete.validate(); err == nil {
		t.Fatal("incomplete custody admitted")
	}
	noPEM := valid
	noPEM.TrustedCertificates = []CertificatePin{{SigningCertURL: "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-abc.pem"}}
	if err := noPEM.validate(); err == nil {
		t.Fatal("pin without retained PEM admitted")
	}
	overPins := valid
	overPins.TrustedCertificates = []CertificatePin{
		{SigningCertURL: "u1", CertificatePEM: "p"}, {SigningCertURL: "u2", CertificatePEM: "p"}, {SigningCertURL: "u3", CertificatePEM: "p"},
	}
	if err := overPins.validate(); err == nil {
		t.Fatal("more than two pins admitted")
	}
}
