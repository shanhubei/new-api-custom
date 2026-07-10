package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/golang-jwt/jwt/v5"
)

const newuserJWTIssuer = "new-api-newuser"

type NewuserJWTClaims struct {
	NewuserId   int `json:"nid"`
	OwnerUserId int `json:"oid"`
	TokenId     int `json:"tid"`
	jwt.RegisteredClaims
}

func newuserJWTSecret() []byte {
	if common.NewuserJWTSecret != "" {
		return []byte(common.NewuserJWTSecret)
	}
	return []byte(common.SessionSecret)
}

func IssueNewuserJWT(newuserId, ownerUserId, tokenId int) (string, error) {
	if newuserId <= 0 || ownerUserId <= 0 || tokenId <= 0 {
		return "", errors.New("invalid newuser jwt payload")
	}
	now := time.Now()
	expireHours := common.NewuserJWTExpireHours
	if expireHours <= 0 {
		expireHours = 168
	}
	claims := NewuserJWTClaims{
		NewuserId:   newuserId,
		OwnerUserId: ownerUserId,
		TokenId:     tokenId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    newuserJWTIssuer,
			Subject:   fmt.Sprintf("%d", newuserId),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(newuserJWTSecret())
}

func ParseNewuserJWT(tokenString string) (*NewuserJWTClaims, error) {
	tokenString = trimBearer(tokenString)
	if tokenString == "" {
		return nil, errors.New("empty token")
	}
	claims := &NewuserJWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return newuserJWTSecret(), nil
	}, jwt.WithIssuer(newuserJWTIssuer))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.NewuserId <= 0 || claims.OwnerUserId <= 0 || claims.TokenId <= 0 {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func trimBearer(auth string) string {
	auth = strings.TrimSpace(auth)
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return auth
}
