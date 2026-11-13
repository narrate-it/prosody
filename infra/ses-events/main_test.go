package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type eventsMonitor struct {
	mu       sync.Mutex
	account  string
	arn      string
	counts   map[string]int
	inputs   map[string]resource.PropertyMap
}

func (m *eventsMonitor) Call(a pulumi.MockCallArgs) (resource.PropertyMap, error) {
	if a.Token != "aws:index/getCallerIdentity:getCallerIdentity" || a.Provider == "" {
		return nil, fmt.Errorf("unexpected or unbound account lookup")
	}
	return resource.NewPropertyMapFromMap(map[string]any{"accountId": m.account, "arn": m.arn}), nil
}

func (m *eventsMonitor) NewResource(a pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if a.TypeToken == "pulumi:pulumi:Stack" {
		return "stack", a.Inputs, nil
	}
	if a.RegisterRPC == nil || a.RegisterRPC.GetVersion() != providerVersion {
		return "", nil, fmt.Errorf("provider version not pinned")
	}
	if a.TypeToken == "pulumi:providers:aws" {
		if a.Inputs["region"].StringValue() != "us-west-2" || len(a.Inputs["allowedAccountIds"].ArrayValue()) != 1 || a.Inputs["allowedAccountIds"].ArrayValue()[0].StringValue() != "123456789012" {
			return "", nil, fmt.Errorf("wrong provider account or region")
		}
		return "provider", a.Inputs, nil
	}
	if !a.RegisterRPC.GetProtect() || !a.RegisterRPC.GetRetainOnDelete() || a.Provider == "" {
		return "", nil, fmt.Errorf("missing protection, retention or explicit provider")
	}
	m.mu.Lock()
	m.counts[a.TypeToken]++
	m.inputs[a.Name] = a.Inputs
	m.mu.Unlock()
	out := a.Inputs.Copy()
	switch a.TypeToken {
	case "aws:sns/topic:Topic":
		if a.Inputs["name"].StringValue() != topicName {
			return "", nil, fmt.Errorf("wrong topic name")
		}
		out["arn"] = resource.NewStringProperty("arn:aws:sns:us-west-2:" + m.account + ":" + topicName)
	case "aws:sns/topicPolicy:TopicPolicy":
		var policy struct {
			Statement []struct {
				Principal map[string]string `json:"Principal"`
				Action    string            `json:"Action"`
				Condition map[string]map[string]string `json:"Condition"`
			} `json:"Statement"`
		}
		if err := json.Unmarshal([]byte(a.Inputs["policy"].StringValue()), &policy); err != nil || len(policy.Statement) != 1 {
			return "", nil, fmt.Errorf("topic policy is not a single bounded statement")
		}
		s := policy.Statement[0]
		if s.Principal["Service"] != "ses.amazonaws.com" || s.Action != "sns:Publish" ||
			s.Condition["StringEquals"]["aws:SourceAccount"] != m.account ||
			s.Condition["ArnLike"]["aws:SourceArn"] != "arn:aws:ses:us-west-2:"+m.account+":configuration-set/"+configurationSet {
			return "", nil, fmt.Errorf("topic policy does not bound the SES source")
		}
	case "aws:sqs/queue:Queue":
		name := a.Inputs["name"].StringValue()
		if name != queueName && name != dlqName {
			return "", nil, fmt.Errorf("unexpected queue")
		}
		out["arn"] = resource.NewStringProperty("arn:aws:sqs:us-west-2:" + m.account + ":" + name)
		out["url"] = resource.NewStringProperty("https://sqs.us-west-2.amazonaws.com/" + m.account + "/" + name)
		if name == queueName {
			if a.Inputs["messageRetentionSeconds"].NumberValue() != 86400 || a.Inputs["receiveWaitTimeSeconds"].NumberValue() != 20 {
				return "", nil, fmt.Errorf("queue retention/polling not bounded to the 24h evidence window")
			}
			if !strings.Contains(a.Inputs["redrivePolicy"].StringValue(), dlqName) {
				return "", nil, fmt.Errorf("queue redrive does not target the review DLQ")
			}
		}
	case "aws:sqs/queuePolicy:QueuePolicy":
		var policy struct {
			Statement []struct {
				Principal map[string]string            `json:"Principal"`
				Action    string                       `json:"Action"`
				Condition map[string]map[string]string `json:"Condition"`
			} `json:"Statement"`
		}
		if err := json.Unmarshal([]byte(a.Inputs["policy"].StringValue()), &policy); err != nil || len(policy.Statement) != 1 {
			return "", nil, fmt.Errorf("queue policy is not a single bounded statement")
		}
		s := policy.Statement[0]
		if s.Principal["Service"] != "sns.amazonaws.com" || s.Action != "sqs:SendMessage" ||
			s.Condition["ArnEquals"]["aws:SourceArn"] != "arn:aws:sns:us-west-2:"+m.account+":"+topicName {
			return "", nil, fmt.Errorf("queue policy does not bound the topic source")
		}
	case "aws:sns/topicSubscription:TopicSubscription":
		if a.Inputs["protocol"].StringValue() != "sqs" || a.Inputs["rawMessageDelivery"].BoolValue() {
			return "", nil, fmt.Errorf("subscription must be SQS with native signatures (no raw stripping)")
		}
	case "aws:sesv2/configurationSetEventDestination:ConfigurationSetEventDestination":
		dest := a.Inputs["eventDestination"].ObjectValue()
		if a.Inputs["configurationSetName"].StringValue() != configurationSet || !dest["enabled"].BoolValue() {
			return "", nil, fmt.Errorf("event destination not bound to the reviewed configuration set")
		}
		types := dest["matchingEventTypes"].ArrayValue()
		if len(types) != 1 || types[0].StringValue() != "SEND" {
			return "", nil, fmt.Errorf("event destination must match SEND only")
		}
	default:
		return "", nil, fmt.Errorf("unexpected resource %s", a.TypeToken)
	}
	return a.Name, out, nil
}

func runMock(t *testing.T, account, arn string) *eventsMonitor {
	t.Helper()
	m := &eventsMonitor{account: account, arn: arn, counts: map[string]int{}, inputs: map[string]resource.PropertyMap{}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return registerEventPath(ctx, account, "us-west-2")
	}, pulumi.WithMocks("prosody-ses-events", "test", m))
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	return m
}

// TestEventGraph checks the exact protected graph and its source policies.
func TestEventGraph(t *testing.T) {
	m := runMock(t, "123456789012", "arn:aws:iam::123456789012:user/prosody-ses")
	want := map[string]int{
		"aws:sns/topic:Topic": 1, "aws:sns/topicPolicy:TopicPolicy": 1,
		"aws:sqs/queue:Queue": 2, "aws:sqs/queuePolicy:QueuePolicy": 1,
		"aws:sns/topicSubscription:TopicSubscription": 1,
		"aws:sesv2/configurationSetEventDestination:ConfigurationSetEventDestination": 1,
	}
	for token, n := range want {
		if m.counts[token] != n {
			t.Errorf("%s: want %d registrations, got %d", token, n, m.counts[token])
		}
	}
	if len(m.counts) != len(want) {
		t.Errorf("unexpected extra resource types: %v", m.counts)
	}
}

// TestRootRefusal rejects the root principal even in the selected account.
func TestRootRefusal(t *testing.T) {
	m := &eventsMonitor{account: "123456789012", arn: "arn:aws:iam::123456789012:root", counts: map[string]int{}, inputs: map[string]resource.PropertyMap{}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return registerEventPath(ctx, "123456789012", "us-west-2")
	}, pulumi.WithMocks("prosody-ses-events", "test", m))
	if err == nil || !strings.Contains(err.Error(), "non-root") {
		t.Fatalf("root principal admitted: %v", err)
	}
}

// TestAccountMismatch rejects a caller from a different account.
func TestAccountMismatch(t *testing.T) {
	m := &eventsMonitor{account: "999999999999", arn: "arn:aws:iam::999999999999:user/x", counts: map[string]int{}, inputs: map[string]resource.PropertyMap{}}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return registerEventPath(ctx, "123456789012", "us-west-2")
	}, pulumi.WithMocks("prosody-ses-events", "test", m))
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("foreign account admitted: %v", err)
	}
}

// TestTarget rejects malformed selections before any registration.
func TestTarget(t *testing.T) {
	for _, c := range []struct{ account, region string }{
		{"", "us-west-2"}, {"123", "us-west-2"}, {"12345678901x", "us-west-2"},
		{"123456789012", ""}, {"123456789012", "us-east-1"}, {"123456789012", "US-WEST-2"},
	} {
		if validateTarget(c.account, c.region) == nil {
			t.Errorf("invalid target accepted: %q %q", c.account, c.region)
		}
	}
	if err := validateTarget("123456789012", "us-west-2"); err != nil {
		t.Fatal(err)
	}
}
