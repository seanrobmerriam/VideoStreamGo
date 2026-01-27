package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/instance"
	"videostreamgo/internal/models/master"
	instanceRepo "videostreamgo/internal/repository/instance"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// Security event types for logging
const (
	SecurityEventCrossServiceToken = "CROSS_SERVICE_TOKEN_ATTEMPT"
	SecurityEventInvalidIssuer     = "INVALID_ISSUER"
	SecurityEventTokenExpired      = "TOKEN_EXPIRED"
	SecurityEventTokenRevoked      = "TOKEN_REVOKED"
)

// logSecurityEvent logs security-related events
func logSecurityEvent(eventType, message, IPAddress string) {
	log.Printf("[SECURITY] type=%s msg=%s ip=%s", eventType, message, IPAddress)
}

// AdminAuthMiddleware handles JWT authentication for platform admins
func AdminAuthMiddleware(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	repo := masterRepo.NewAdminRepository(db)
	return func(c *gin.Context) {
		IPAddress := c.ClientIP()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"UNAUTHORIZED",
				"Authorization header is required",
				"",
			))
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_TOKEN_FORMAT",
				"Invalid authorization header format. Use 'Bearer <token>'",
				"",
			))
			return
		}

		tokenString := parts[1]

		// First, parse without verification to check issuer claim
		token, _ := jwt.Parse(tokenString, nil)
		if token != nil {
			claims, ok := token.Claims.(jwt.MapClaims)
			if ok {
				issuer, _ := claims["iss"].(string)
				// Check for cross-service token attempt
				if issuer != "" && issuer != config.ServiceIdentifierPlatform {
					logSecurityEvent(
						SecurityEventCrossServiceToken,
						"Cross-service token attempt detected",
						IPAddress,
					)
					c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
						"INVALID_ISSUER",
						"Token issued by incorrect service",
						"Expected: "+config.ServiceIdentifierPlatform,
					))
					return
				}
			}
		}

		// Parse and validate JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(cfg.App.PlatformJWTSecret), nil
		})

		if err != nil {
			logSecurityEvent(
				SecurityEventTokenExpired,
				"Invalid or expired token: "+err.Error(),
				IPAddress,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_TOKEN",
				"Invalid or expired token",
				err.Error(),
			))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_CLAIMS",
				"Invalid token claims",
				"",
			))
			return
		}

		// Validate issuer claim
		issuer, _ := claims["iss"].(string)
		if issuer != config.ServiceIdentifierPlatform {
			logSecurityEvent(
				SecurityEventInvalidIssuer,
				"Token with invalid issuer: "+issuer,
				IPAddress,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_ISSUER",
				"Token issuer is not valid for this service",
				"",
			))
			return
		}

		// Extract admin ID from claims
		adminIDStr, ok := claims["admin_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"MISSING_ADMIN_ID",
				"Token is missing admin_id claim",
				"",
			))
			return
		}

		adminID, err := uuid.Parse(adminIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_ADMIN_ID",
				"Token contains invalid admin_id",
				"",
			))
			return
		}

		// Fetch admin user from database
		admin, err := repo.GetByID(c.Request.Context(), adminID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"ADMIN_NOT_FOUND",
				"Admin user not found",
				"",
			))
			return
		}

		if admin.Status != master.AdminStatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse(
				"ADMIN_INACTIVE",
				"Admin account is not active",
				"",
			))
			return
		}

		// Set admin user in context
		c.Set(string(types.ContextKeyAdminUser), admin)
		c.Set(string(types.ContextKeyAdminID), adminID)

		c.Next()
	}
}

// RequireRole middleware checks if admin has required role
func RequireRole(allowedRoles ...master.AdminRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, exists := c.Get(string(types.ContextKeyAdminUser))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"NOT_AUTHENTICATED",
				"Admin user not found in context",
				"",
			))
			return
		}

		adminUser, ok := admin.(*master.AdminUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, types.ErrorResponse(
				"INVALID_ADMIN_TYPE",
				"Invalid admin user type in context",
				"",
			))
			return
		}

		// Super admins have access to everything
		if adminUser.Role == master.AdminRoleSuperAdmin {
			c.Next()
			return
		}

		// Check if role is allowed
		for _, role := range allowedRoles {
			if adminUser.Role == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse(
			"INSUFFICIENT_PERMISSIONS",
			"You do not have permission to perform this action",
			"",
		))
	}
}

// GenerateAdminToken generates a JWT token for an admin user
func GenerateAdminToken(admin *master.AdminUser, cfg *config.Config) (string, string, error) {
	jti := uuid.New().String()
	sessionID := uuid.New().String()

	claims := jwt.MapClaims{
		"admin_id": admin.ID.String(),
		"email":    admin.Email,
		"role":     string(admin.Role),
		"iss":      config.ServiceIdentifierPlatform,
		"jti":      jti,
		"sid":      sessionID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.App.PlatformJWTSecret))
	return tokenString, jti, err
}

// InstanceAuthMiddleware handles JWT authentication for instance users
func InstanceAuthMiddleware(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	repo := instanceRepo.NewUserRepository(db)
	return func(c *gin.Context) {
		IPAddress := c.ClientIP()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"UNAUTHORIZED",
				"Authorization header is required",
				"",
			))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_TOKEN_FORMAT",
				"Invalid authorization header format. Use 'Bearer <token>'",
				"",
			))
			return
		}

		tokenString := parts[1]

		// First, parse without verification to check issuer claim
		token, _ := jwt.Parse(tokenString, nil)
		if token != nil {
			claims, ok := token.Claims.(jwt.MapClaims)
			if ok {
				issuer, _ := claims["iss"].(string)
				// Check for cross-service token attempt
				if issuer != "" && issuer != config.ServiceIdentifierInstance {
					logSecurityEvent(
						SecurityEventCrossServiceToken,
						"Cross-service token attempt detected",
						IPAddress,
					)
					c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
						"INVALID_ISSUER",
						"Token issued by incorrect service",
						"Expected: "+config.ServiceIdentifierInstance,
					))
					return
				}
			}
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return []byte(cfg.App.InstanceJWTSecret), nil
		})

		if err != nil {
			logSecurityEvent(
				SecurityEventTokenExpired,
				"Invalid or expired token: "+err.Error(),
				IPAddress,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_TOKEN",
				"Invalid or expired token",
				err.Error(),
			))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_CLAIMS",
				"Invalid token claims",
				"",
			))
			return
		}

		// Validate issuer claim
		issuer, _ := claims["iss"].(string)
		if issuer != config.ServiceIdentifierInstance {
			logSecurityEvent(
				SecurityEventInvalidIssuer,
				"Token with invalid issuer: "+issuer,
				IPAddress,
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_ISSUER",
				"Token issuer is not valid for this service",
				"",
			))
			return
		}

		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"MISSING_USER_ID",
				"Token is missing user_id claim",
				"",
			))
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_USER_ID",
				"Token contains invalid user_id",
				"",
			))
			return
		}

		instanceIDStr, ok := claims["instance_id"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"MISSING_INSTANCE_ID",
				"Token is missing instance_id claim",
				"",
			))
			return
		}

		instanceID, err := uuid.Parse(instanceIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"INVALID_INSTANCE_ID",
				"Token contains invalid instance_id",
				"",
			))
			return
		}

		// Fetch user from database
		user, err := repo.GetByID(c.Request.Context(), userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse(
				"USER_NOT_FOUND",
				"User not found",
				"",
			))
			return
		}

		if user.Status != instance.UserStatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse(
				"USER_INACTIVE",
				"User account is not active",
				"",
			))
			return
		}

		// Set user in context
		c.Set(string(types.ContextKeyUser), user)
		c.Set(string(types.ContextKeyUserID), userID)
		c.Set(string(types.ContextKeyInstanceID), instanceID)

		c.Next()
	}
}

// GenerateUserToken generates a JWT token for an instance user
func GenerateUserToken(user *instance.User, instanceID uuid.UUID, cfg *config.Config) (string, string, error) {
	jti := uuid.New().String()
	sessionID := uuid.New().String()

	claims := jwt.MapClaims{
		"user_id":     user.ID.String(),
		"instance_id": instanceID.String(),
		"username":    user.Username,
		"email":       user.Email,
		"role":        string(user.Role),
		"iss":         config.ServiceIdentifierInstance,
		"jti":         jti,
		"sid":         sessionID,
		"exp":         time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.App.InstanceJWTSecret))
	return tokenString, jti, err
}
