package webpush

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GetVAPIDAuthorizationHeader creates and signs a VAPID JWT for a Web Push request.
//
// The token includes standard VAPID claims:
//   - aud: derived from the endpoint origin (<scheme>://<host>)
//   - exp: UNIX expiration timestamp from the provided expiration time
//   - sub: VAPID subject, normalized to a valid mailto/http/https format
//
// It signs the JWT using ES256 with the private key from vapidInfo and returns
// the serialized token string, or an error if endpoint parsing, key retrieval,
// or token signing fails.
func GetVAPIDAuthorizationHeader(
	endpoint string,
	vapidInfo Vapid,
	expiration time.Time,
) (signedToken string, err error) {
	subURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"aud": fmt.Sprintf("%s://%s", subURL.Scheme, subURL.Host),
		"exp": expiration.Unix(),
		"sub": formatVapidSubject(vapidInfo.Subject()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	privSignKey, err := vapidInfo.ECDSAPrivKey()
	if err != nil {
		return "", err
	}
	jwtString, err := token.SignedString(privSignKey)
	if err != nil {
		return "", err
	}

	return jwtString, nil
}

func formatVapidSubject(subject string) string {
	if strings.HasPrefix(subject, "mailto:") || strings.HasPrefix(subject, "https://") || strings.HasPrefix(subject, "http://") {
		return subject
	}
	return fmt.Sprintf("mailto:%s", subject)
}
