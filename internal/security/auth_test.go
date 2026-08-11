package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/udaypt/trading-app/config"
)

func TestHashPassword_CheckPasswordHash(t *testing.T) {
	t.Run("correct password verifies against its hash", func(t *testing.T) {
		hash, err := HashPassword("s3cr3t-password")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, "s3cr3t-password", hash)
		assert.True(t, CheckPasswordHash("s3cr3t-password", hash))
	})

	t.Run("wrong password fails verification", func(t *testing.T) {
		hash, err := HashPassword("s3cr3t-password")
		require.NoError(t, err)
		assert.False(t, CheckPasswordHash("wrong-password", hash))
	})

	t.Run("malformed hash fails verification instead of panicking", func(t *testing.T) {
		assert.False(t, CheckPasswordHash("anything", "not-a-bcrypt-hash"))
	})
}

func TestGenerateJWT_ValidateJWT(t *testing.T) {
	t.Run("round trips claims", func(t *testing.T) {
		token, err := GenerateJWT(7, "user@example.com")
		require.NoError(t, err)
		require.NotEmpty(t, token)

		claims, err := ValidateJWT(token)
		require.NoError(t, err)
		assert.Equal(t, int64(7), claims.UserID)
		assert.Equal(t, "user@example.com", claims.Email)
		assert.Equal(t, "market-app", claims.Issuer)
	})

	t.Run("garbage token is rejected", func(t *testing.T) {
		_, err := ValidateJWT("not-a-jwt")
		assert.Error(t, err)
	})

	t.Run("token signed with a different key is rejected", func(t *testing.T) {
		claims := &Claims{
			UserID: 1,
			Email:  "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte("a-different-key"))
		require.NoError(t, err)

		_, err = ValidateJWT(signed)
		assert.Error(t, err)
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		claims := &Claims{
			UserID: 1,
			Email:  "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(config.JWT_SECRET))
		require.NoError(t, err)

		_, err = ValidateJWT(signed)
		assert.Error(t, err)
	})

	t.Run("token with unexpected signing method is rejected", func(t *testing.T) {
		claims := &Claims{
			UserID: 1,
			Email:  "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = ValidateJWT(signed)
		assert.Error(t, err)
	})
}
