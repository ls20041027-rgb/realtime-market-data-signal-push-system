package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"push_gateway/model"
	"push_gateway/storage"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// JWT secret key (in production should come from config/env)
const jwtSecret = "tornado-seeker-jwt-secret-key"
const jwtExpireHours = 72

func handleRegister(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, model.Err(model.CodeInvalidParam, "invalid params: "+err.Error()))
			return
		}

		req.Username = strings.TrimSpace(req.Username)

		// Check if user already exists
		existing, _ := d.MySQL.QueryUserByUsername(c.Request.Context(), req.Username)
		if existing != nil {
			c.JSON(http.StatusConflict, model.Err(model.CodeInvalidParam, "username already exists"))
			return
		}

		// Hash password
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeError(c, err)
			return
		}

		// Create user
		user, err := d.MySQL.CreateUser(c.Request.Context(), req.Username, string(hashed))
		if err != nil {
			writeError(c, err)
			return
		}

		// Generate token
		token, err := generateToken(user.ID, user.Username)
		if err != nil {
			writeError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.Ok(AuthResponse{
			Token:    token,
			Username: user.Username,
		}))
	}
}

func handleLogin(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, model.Err(model.CodeInvalidParam, "invalid params: "+err.Error()))
			return
		}

		req.Username = strings.TrimSpace(req.Username)

		// Find user
		user, err := d.MySQL.QueryUserByUsername(c.Request.Context(), req.Username)
		if err != nil {
			if err == storage.ErrNotFound {
				c.JSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "invalid username or password"))
				return
			}
			writeError(c, err)
			return
		}

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "invalid username or password"))
			return
		}

		// Generate token
		token, err := generateToken(user.ID, user.Username)
		if err != nil {
			writeError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.Ok(AuthResponse{
			Token:    token,
			Username: user.Username,
		}))
	}
}

func handleGetMe(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "unauthorized"))
			return
		}

		user, err := d.MySQL.QueryUserByID(c.Request.Context(), userID.(int64))
		if err != nil {
			writeError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.Ok(gin.H{
			"id":       user.ID,
			"username": user.Username,
		}))
	}
}

func generateToken(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * jwtExpireHours).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "invalid authorization format"))
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "invalid or expired token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Err(model.CodeInvalidParam, "invalid token claims"))
			return
		}

		userID, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)
		c.Set("user_id", int64(userID))
		c.Set("username", username)
		c.Next()
	}
}
