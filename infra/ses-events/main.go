// Command ses-events declares the Prosody SES verified-event path: a dedicated
// SNS topic whose publication policy admits only the intended SES
// configuration-set source, an SQS subscription retaining native SNS
// signatures (no raw-message stripping), a dead-letter queue for review, and
// the SES Send event destination. It creates no credentials and sends no mail.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	providerVersion      = "7.54.0"
	supportedRegion      = "us-west-2"
	configurationSet     = "prosody-identity"
	topicName            = "prosody-ses-events"
	queueName            = "prosody-ses-events"
	dlqName              = "prosody-ses-events-dlq"
	eventDestinationName = "prosody-sns"
)

var accountPattern = regexp.MustCompile(`^[0-9]{12}$`)

type awsProvider struct{ pulumi.ProviderResourceState }
type topicState struct {
	pulumi.CustomResourceState
	Arn pulumi.StringOutput `pulumi:"arn"`
}
type queueState struct {
	pulumi.CustomResourceState
	Arn pulumi.StringOutput `pulumi:"arn"`
	URL pulumi.StringOutput `pulumi:"url"`
}

// main requires an explicit account and region; preview and apply are separate effects.
func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		c := config.New(ctx, "")
		return registerEventPath(ctx, c.Require("accountId"), c.Require("region"))
	})
}

// validateTarget rejects absent or noncanonical account/region selections.
func validateTarget(account, region string) error {
	if !accountPattern.MatchString(account) || region != supportedRegion {
		return fmt.Errorf("an explicit AWS account and supported us-west-2 region are required")
	}
	return nil
}

// topicPolicy admits only the SES service publishing for this account and the
// exact reviewed configuration set. A signed SNS envelope authenticates the
// topic; this source policy is the separate publisher gate.
func topicPolicy(account, topicARN string) string {
	return fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SesEventPublishOnly",
      "Effect": "Allow",
      "Principal": {"Service": "ses.amazonaws.com"},
      "Action": "sns:Publish",
      "Resource": %q,
      "Condition": {
        "StringEquals": {"aws:SourceAccount": %q},
        "ArnLike": {"aws:SourceArn": %q}
      }
    }
  ]
}`, topicARN, account, "arn:aws:ses:"+supportedRegion+":"+account+":configuration-set/"+configurationSet)
}

// queuePolicy admits only the dedicated topic as the queue's SNS source.
func queuePolicy(queueARN, topicARN string) string {
	return fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SnsTopicSendOnly",
      "Effect": "Allow",
      "Principal": {"Service": "sns.amazonaws.com"},
      "Action": "sqs:SendMessage",
      "Resource": %q,
      "Condition": {"ArnEquals": {"aws:SourceArn": %q}}
    }
  ]
}`, queueARN, topicARN)
}

// registerEventPath checks account admission before declaring the protected
// event graph. Native signatures are preserved end to end: the subscription
// must not strip raw messages, or AMOS verification can never succeed.
func registerEventPath(ctx *pulumi.Context, account, region string) error {
	if err := validateTarget(account, region); err != nil {
		return err
	}
	var provider awsProvider
	if err := ctx.RegisterResource("pulumi:providers:aws", "ses-events-account", pulumi.Map{
		"region": pulumi.String(region), "allowedAccountIds": pulumi.StringArray{pulumi.String(account)},
	}, &provider, pulumi.Version(providerVersion)); err != nil {
		return err
	}
	var caller struct {
		AccountID string `pulumi:"accountId"`
		Arn       string `pulumi:"arn"`
	}
	if err := ctx.Invoke("aws:index/getCallerIdentity:getCallerIdentity", map[string]any{}, &caller, pulumi.Provider(&provider), pulumi.Version(providerVersion)); err != nil {
		return fmt.Errorf("read SES account: %w", err)
	}
	if caller.AccountID != account {
		return fmt.Errorf("SES account does not match selected account")
	}
	if strings.HasSuffix(caller.Arn, ":root") {
		return fmt.Errorf("SES event-path apply requires a non-root principal")
	}
	opts := []pulumi.ResourceOption{pulumi.Provider(&provider), pulumi.Version(providerVersion), pulumi.Protect(true), pulumi.RetainOnDelete(true)}

	var topic topicState
	if err := ctx.RegisterResource("aws:sns/topic:Topic", "prosody-ses-events-topic", pulumi.Map{
		"name": pulumi.String(topicName),
	}, &topic, opts...); err != nil {
		return err
	}
	if err := ctx.RegisterResource("aws:sns/topicPolicy:TopicPolicy", "prosody-ses-events-topic-policy", pulumi.Map{
		"arn":      topic.Arn,
		"policy":   topic.Arn.ApplyT(func(arn string) (string, error) { return topicPolicy(account, arn), nil }).(pulumi.StringOutput),
	}, &pulumi.CustomResourceState{}, append(opts, pulumi.DependsOn([]pulumi.Resource{&topic}))...); err != nil {
		return err
	}
	var dlq queueState
	if err := ctx.RegisterResource("aws:sqs/queue:Queue", "prosody-ses-events-dlq", pulumi.Map{
		"name":                     pulumi.String(dlqName),
		"messageRetentionSeconds":  pulumi.Int(1209600),
		"visibilityTimeoutSeconds": pulumi.Int(60),
	}, &dlq, opts...); err != nil {
		return err
	}
	var queue queueState
	if err := ctx.RegisterResource("aws:sqs/queue:Queue", "prosody-ses-events-queue", pulumi.Map{
		"name":                       pulumi.String(queueName),
		"messageRetentionSeconds":    pulumi.Int(86400),
		"visibilityTimeoutSeconds":   pulumi.Int(60),
		"receiveWaitTimeSeconds": pulumi.Int(20),
		"redrivePolicy": dlq.Arn.ApplyT(func(arn string) (string, error) {
			return fmt.Sprintf(`{"deadLetterTargetArn":%q,"maxReceiveCount":5}`, arn), nil
		}).(pulumi.StringOutput),
	}, &queue, append(opts, pulumi.DependsOn([]pulumi.Resource{&dlq}))...); err != nil {
		return err
	}
	if err := ctx.RegisterResource("aws:sqs/queuePolicy:QueuePolicy", "prosody-ses-events-queue-policy", pulumi.Map{
		"queueUrl": queue.URL,
		"policy": pulumi.All(queue.Arn, topic.Arn).ApplyT(func(v []any) (string, error) {
			return queuePolicy(v[0].(string), v[1].(string)), nil
		}).(pulumi.StringOutput),
	}, &pulumi.CustomResourceState{}, append(opts, pulumi.DependsOn([]pulumi.Resource{&queue, &topic}))...); err != nil {
		return err
	}
	if err := ctx.RegisterResource("aws:sns/topicSubscription:TopicSubscription", "prosody-ses-events-subscription", pulumi.Map{
		"topic":              topic.Arn,
		"protocol":           pulumi.String("sqs"),
		"endpoint":           queue.Arn,
		"rawMessageDelivery": pulumi.Bool(false),
	}, &pulumi.CustomResourceState{}, append(opts, pulumi.DependsOn([]pulumi.Resource{&topic, &queue}))...); err != nil {
		return err
	}
	if err := ctx.RegisterResource("aws:sesv2/configurationSetEventDestination:ConfigurationSetEventDestination", "prosody-ses-send-events", pulumi.Map{
		"configurationSetName": pulumi.String(configurationSet),
		"eventDestinationName": pulumi.String(eventDestinationName),
		"eventDestination": pulumi.Map{
			"enabled":            pulumi.Bool(true),
			"matchingEventTypes": pulumi.StringArray{pulumi.String("SEND")},
			"snsDestination":     pulumi.Map{"topicArn": topic.Arn},
		},
	}, &pulumi.CustomResourceState{}, append(opts, pulumi.DependsOn([]pulumi.Resource{&topic}))...); err != nil {
		return err
	}

	ctx.Export("topicArn", topic.Arn)
	ctx.Export("queueUrl", queue.URL)
	ctx.Export("queueArn", queue.Arn)
	ctx.Export("deadLetterQueueUrl", dlq.URL)
	ctx.Export("eventDestination", pulumi.String(eventDestinationName))
	ctx.Export("status", pulumi.String("event-path-declared-not-qualified"))
	return nil
}
