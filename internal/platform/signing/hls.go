package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Signer struct {
	Secret []byte
}

type Signed struct {
	URL string
	Exp int64
	UID string
	Sig string
}

func New(secret string) *Signer {
	return &Signer{Secret: []byte(secret)}
}

func (s *Signer) Sign(rawURL, userID string, exp time.Time) Signed {
	sig := s.signValue(rawURL, userID, exp.Unix())
	return Signed{URL: rawURL, Exp: exp.Unix(), UID: userID, Sig: sig}
}

func (s *Signer) Verify(rawURL, userID string, exp int64, sig string) bool {
	if time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.signValue(rawURL, userID, exp)))
}

func (s *Signer) signValue(rawURL, userID string, exp int64) string {
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(rawURL))
	mac.Write([]byte("|"))
	mac.Write([]byte(userID))
	mac.Write([]byte("|"))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func BuildSignedURL(base string, signed Signed) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("url", signed.URL)
	q.Set("exp", strconv.FormatInt(signed.Exp, 10))
	q.Set("uid", signed.UID)
	q.Set("sig", signed.Sig)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func ExtractSigned(query url.Values) (string, string, int64, string, error) {
	rawURL := strings.TrimSpace(query.Get("url"))
	uid := strings.TrimSpace(query.Get("uid"))
	expStr := strings.TrimSpace(query.Get("exp"))
	sig := strings.TrimSpace(query.Get("sig"))
	if rawURL == "" || uid == "" || expStr == "" || sig == "" {
		return "", "", 0, "", fmt.Errorf("missing signed params")
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return "", "", 0, "", err
	}
	return rawURL, uid, exp, sig, nil
}
