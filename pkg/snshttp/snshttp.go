package snshttp

import (
	"net/http"
	"time"

	"gotoexec/pkg/utils"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type OnSNSNotificationFn = func(c *gin.Context, notification *SNSNotification) error

const (
	// The connection will time out in 15 seconds, so we need to cut our processing before that moment
	// https://docs.amazonaws.cn/en_us/sns/latest/dg/SendMessageToHttp.prepare.html
	requestTimeout = 14 * time.Second

	// Max time we will spend in activities like trying to verify the sns signature, etc..
	maxHTTPClientTimeout = 10 * time.Second
)

type SNSHandler struct {
	credentials *authOption
	httpClient  *http.Client

	certCache         *utils.Cache
	certCacheDuration time.Duration
}

func NewSNSHTTPHandler(opts ...Option) *SNSHandler {
	handler := &SNSHandler{
		certCache: utils.NewCache(),
	}

	for _, opt := range opts {
		opt.apply(handler)
	}

	if handler.httpClient == nil {
		// By default, we will use a retryable client
		client := retryablehttp.NewClient()
		client.RetryWaitMax = maxHTTPClientTimeout
		handler.httpClient = client.StandardClient()
	}

	return handler
}

func (h *SNSHandler) GetSNSRequestHandler(onMessageFn OnSNSNotificationFn) gin.HandlerFunc {
	innerHandler := func(c *gin.Context) (interface{}, error) {
		if !h.credentials.Check(c.Request) {
			c.Header("WWW-Authenticate", `Basic realm="ses"`)
			return nil, &utils.RequestError{StatusCode: http.StatusUnauthorized, Err: errors.New("Unauthorized")}
		}

		// Read the body
		notification := new(SNSNotification)
		{
			err := c.ShouldBindWith(notification, binding.JSON)
			if err != nil {
				return nil, errors.WithMessage(err, "cannot bind sns notification data")
			}
		}

		snsMessageType := c.GetHeader("X-Amz-Sns-Message-Type")

		// Use the Type header so we can avoid parsing the body unless we know it's
		// an event we support.
		switch snsMessageType {

		// Notifications should be the most common case and switch statements are
		// checked in definition order.
		case "Notification":
			return nil, h.handleNotification(c, notification, onMessageFn)
		case "SubscriptionConfirmation":
			return nil, h.handleSubscriptionConfirmation(notification)
		default:
			return nil, errors.Errorf("unsupported notification type %s", snsMessageType)
		}
	}

	return timeout.New(
		timeout.WithTimeout(requestTimeout),
		timeout.WithHandler(utils.WrapRequest(innerHandler)),
	)
}

func (h *SNSHandler) handleSubscriptionConfirmation(notification *SNSNotification) error {

	if err := h.verifyNotificationSignature(notification); err != nil {
		return errors.WithMessage(err, "failed to verify sns notification signature")
	}

	req, err := http.NewRequest("GET", notification.SubscribeURL, nil)
	if err != nil {
		return errors.WithMessage(err, "failed to create sns confirm subscription request")
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "failed to execute sns confirm subscription request")
	}
	defer resp.Body.Close()

	// Server is expected to return 200 OK but we can treat any 200 level code as
	// success.
	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		return errors.Errorf("unexpected status code %d for sns confirm subscription request", resp.StatusCode)
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logFields := utils.MergeMap(notification.LogFields(), logrus.Fields{
			"snsSignature":        notification.Signature,
			"snsSignatureVersion": notification.SignatureVersion,
			"snsSigningCertURL":   notification.SigningCertURL,
			"snsSubscribeURL":     notification.SubscribeURL,
		})
		logrus.WithFields(logFields).Debug("sns subscription confirmed")
	} else {
		logrus.WithFields(notification.LogFields()).Info("sns subscription confirmed")
	}

	return nil
}

func (h *SNSHandler) handleNotification(c *gin.Context, notification *SNSNotification, onMessageFn OnSNSNotificationFn) error {
	if err := h.verifyNotificationSignature(notification); err != nil {
		return errors.WithMessage(err, "failed to verify sns notification signature")
	}

	// Default the subject to the truncated ARN
	if notification.Subject == "" {
		notification.Subject = notification.ARNShort()
	}

	if err := onMessageFn(c, notification); err != nil {
		return errors.WithMessage(err, "failed to process sns notification")
	}

	return nil
}
