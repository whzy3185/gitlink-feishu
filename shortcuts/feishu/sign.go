package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// SignCustomBotRequest implements the Feishu custom bot signature algorithm.
// Feishu uses timestamp + "\n" + secret as the HMAC key and signs an empty body.
func SignCustomBotRequest(timestamp int64, secret string) string {
	stringToSign := strconv.FormatInt(timestamp, 10) + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func timestampSeconds(now time.Time) int64 {
	return now.Unix()
}
