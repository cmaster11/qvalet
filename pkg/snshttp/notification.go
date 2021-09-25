package snshttp

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// https://github.com/robbiet480/go.sns/issues/2
var hostPattern = regexp.MustCompile(`^sns\.[a-zA-Z0-9\-]{3,}\.amazonaws\.com(\.cn)?$`)

const SNSNotificationOriginKeyTopicArn = "snsTopicArn"
const SNSNotificationOriginKeyMessageId = "snsMessageId"

/*
{
  "Type": "Notification",
  "MessageId": "6d1b8a6e-25f5-504f-b1f4-d297c80b0c33",
  "TopicArn": "arn:aws:sns:us-east-1:661666139333:testwebhookio",
  "Subject": "teee",
  "Message": "eeeee",
  "Timestamp": "2021-04-21T09:09:18.710Z",
  "SignatureVersion": "1",
  "Signature": "PIwk/6jFac+wL5YSZKrTYrSAyH9+ZFrHWShjWEzDFPs/eziSw89Np2ORBamRI1XbQ3wBgW2LT6ykNHkCPjpGwXPHT+QW8MMDErj4UNueMpI8T9kBGyfGB7W6HJzMTOAlanV+xdXR4jRroy451/c0COCroSPonu9RwO5ReyHVIBr8OoPr89+LvVAytd7y5cU/8lnJiauvpwQ6bpOIlRvBQkDvhyzVJ91rSw0KwR7tyTfZGfED1ouMmm46tv7nJ3befqfHnpG7fhBbR+T8tUQt3F6ijuAtEy3ljNn2P4GTy2Om78H3SyE0kbmNz8xSqmOAqSTORnt6SyG0Q8cNhjqdqw==",
  "SigningCertURL": "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-010a507c1833636cd94bdb98bd93083a.pem",
  "UnsubscribeURL": "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:661666139333:testwebhookio:1fcd3960-bb69-4c4f-a160-c0917206ad1b",
  "MessageAttributes": {
    "str": {
      "Type": "String",
      "Value": "asd"
    },
    "bin": {
      "Type": "Binary",
      "Value": "YXNkYXNk"
    },
    "strarrrrrr": {
      "Type": "String.Array",
      "Value": "sssss"
    },
    "num": {
      "Type": "Number",
      "Value": "33"
    }
  }
}
*/

// @formatter:off
/// [sns-notification]
// Notification events are sent for messages that are published to the SNS
// topic.
type SNSNotification struct {
	Subject           string                      `json:"Subject"`
	Message           string                      `json:"Message"`
	MessageId         string                      `json:"MessageId"`
	Signature         string                      `json:"Signature"`
	SignatureVersion  string                      `json:"SignatureVersion"`
	SigningCertURL    string                      `json:"SigningCertURL"`
	SubscribeURL      string                      `json:"SubscribeURL"`
	Timestamp         string                      `json:"Timestamp"`
	TopicArn          string                      `json:"TopicArn"`
	Token             string                      `json:"Token"`
	Type              string                      `json:"Type"`
	UnsubscribeURL    string                      `json:"UnsubscribeURL"`
	MessageAttributes map[string]MessageAttribute `json:"MessageAttributes"`
}

/*
	"myString": {
	  "Type": "String",
	  "Value": "Hello!"
	},
*/
// See https://docs.aws.amazon.com/sns/latest/dg/sns-message-attributes.html
type MessageAttribute struct {
	Type  string      `json:"Type"`
	Value interface{} `json:"Value"`
}

/// [sns-notification]
// @formatter:on

func (notification *SNSNotification) ARNShort() string {
	return notification.TopicArn[strings.LastIndex(notification.TopicArn, ":")+1:]
}

func (notification *SNSNotification) LogFields() logrus.Fields {
	return logrus.Fields{
		"snsMessageType": notification.Type,
		"snsMessageId":   notification.MessageId,
	}
}

// https://github.com/robbiet480/go.sns/blob/master/main.go#L51
func (notification *SNSNotification) buildSignature() []byte {
	var builtSignature bytes.Buffer
	signableKeys := []string{"Message", "MessageId", "Subject", "SubscribeURL", "Timestamp", "Token", "TopicArn", "Type"}
	for _, key := range signableKeys {
		reflectedStruct := reflect.ValueOf(notification)
		field := reflect.Indirect(reflectedStruct).FieldByName(key)
		value := field.String()
		if field.IsValid() && value != "" {
			builtSignature.WriteString(key + "\n")
			builtSignature.WriteString(value + "\n")
		}
	}
	return builtSignature.Bytes()
}

// verifyNotificationSignature will verify that a payload came from SNS
func (h *SNSHandler) verifyNotificationSignature(notification *SNSNotification) error {
	payloadSignature, err := base64.StdEncoding.DecodeString(notification.Signature)
	if err != nil {
		return errors.WithMessage(err, "failed to decode notification signature")
	}

	certURL, err := url.Parse(notification.SigningCertURL)
	if err != nil {
		return errors.WithMessage(err, "failed to parse cert signing url")
	}

	if certURL.Scheme != "https" {
		return errors.New("sns cert url should be using https scheme")
	}

	if !hostPattern.Match([]byte(certURL.Host)) {
		return errors.New("sns cert url is using an invalid domain")
	}

	var body []byte
	if bodyIntf := h.certCache.Get(notification.SigningCertURL); bodyIntf != nil {
		_body, ok := bodyIntf.([]byte)
		if !ok {
			logrus.Error("cached sns cert body could no be cast to []byte")
		} else {
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.Debug("found cached sns cert")
			}
			body = _body
		}
	}

	if len(body) == 0 {
		resp, err := h.httpClient.Get(notification.SigningCertURL)
		if err != nil {
			return errors.WithMessage(err, "failed to get signing certificate")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.WithMessagef(err, "failed to get signing certificate (bad status code %d)", resp.StatusCode)
		}

		_body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.WithMessage(err, "failed to read signing certificate body")
		}

		if h.certCacheDuration > 0 {
			h.certCache.SetWithDuration(notification.SigningCertURL, _body, h.certCacheDuration)
		}

		body = _body
	}

	decodedPem, _ := pem.Decode(body)
	if decodedPem == nil {
		return errors.New("failed to decode signing certificate")
	}

	parsedCertificate, err := x509.ParseCertificate(decodedPem.Bytes)
	if err != nil {
		return errors.WithMessage(err, "failed to parse signing certificate")
	}

	if err := parsedCertificate.CheckSignature(x509.SHA1WithRSA, notification.buildSignature(), payloadSignature); err != nil {
		return errors.WithMessage(err, "failed to check signature")
	}

	return nil
}
