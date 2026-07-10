package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func NewuserModuleEnabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.NewuserEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "newuser module is disabled",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func NewuserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.NewuserEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "newuser module is disabled",
			})
			c.Abort()
			return
		}
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "authorization required",
			})
			c.Abort()
			return
		}
		claims, err := service.ParseNewuserJWT(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}
		nu, err := model.GetNewuserById(claims.NewuserId)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "user not found",
			})
			c.Abort()
			return
		}
		if nu.Status != common.UserStatusEnabled {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "user disabled",
			})
			c.Abort()
			return
		}
		if nu.OwnerUserId != claims.OwnerUserId || nu.TokenId != claims.TokenId {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "token mismatch",
			})
			c.Abort()
			return
		}
		c.Set("newuser_id", nu.Id)
		c.Set("newuser_owner_id", nu.OwnerUserId)
		c.Set("newuser_token_id", nu.TokenId)
		c.Set("newuser", nu)
		c.Next()
	}
}

func NewuserOrgOwnerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.NewuserEnabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "newuser module is disabled",
			})
			c.Abort()
			return
		}
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "authorization required",
			})
			c.Abort()
			return
		}
		claims, err := service.ParseNewuserJWT(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}
		nu, err := model.GetNewuserById(claims.NewuserId)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "user not found",
			})
			c.Abort()
			return
		}
		if nu.Status != common.UserStatusEnabled {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "user disabled",
			})
			c.Abort()
			return
		}
		if !nu.IsOrgOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "organization owner permission required",
			})
			c.Abort()
			return
		}
		if nu.OwnerUserId != claims.OwnerUserId || nu.TokenId != claims.TokenId {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "token mismatch",
			})
			c.Abort()
			return
		}
		c.Set("newuser_id", nu.Id)
		c.Set("newuser_owner_id", nu.OwnerUserId)
		c.Set("newuser_token_id", nu.TokenId)
		c.Set("newuser", nu)
		c.Next()
	}
}

func ResolveNewuserOwnerId(c *gin.Context, bodyOwnerId int) (int, error) {
	if bodyOwnerId > 0 {
		return bodyOwnerId, nil
	}
	if common.NewuserDefaultOwnerId > 0 {
		return common.NewuserDefaultOwnerId, nil
	}
	headerVal := c.GetHeader("New-Api-Org-Owner")
	if headerVal == "" {
		return 0, errors.New("owner_user_id is required")
	}
	ownerId, err := strconv.Atoi(headerVal)
	if err != nil || ownerId <= 0 {
		return 0, errors.New("invalid owner_user_id")
	}
	return ownerId, nil
}
