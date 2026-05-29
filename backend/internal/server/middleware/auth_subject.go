package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthSubject is the minimal authenticated identity stored in gin context.
type AuthSubject struct {
	UserID        int64
	Concurrency   int
	PrincipalID   string
	PrincipalType string
	IsSystem      bool
}

func (s AuthSubject) ActorUserID() int64 {
	if s.IsSystem || s.UserID <= 0 {
		return 0
	}
	return s.UserID
}

func (s AuthSubject) HumanUserID() (int64, bool) {
	if s.IsSystem || s.UserID <= 0 {
		return 0, false
	}
	return s.UserID, true
}

func GetAuthSubjectFromContext(c *gin.Context) (AuthSubject, bool) {
	value, exists := c.Get(string(ContextKeyUser))
	if !exists {
		return AuthSubject{}, false
	}
	subject, ok := value.(AuthSubject)
	return subject, ok
}

func GetUserRoleFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

func (s AuthSubject) ActorScope(prefix string) string {
	if s.IsSystem {
		id := strings.TrimSpace(s.PrincipalID)
		if id == "" {
			id = "system"
		}
		return prefix + ":" + id
	}
	if s.UserID > 0 {
		return prefix + ":" + strconv.FormatInt(s.UserID, 10)
	}
	if id := strings.TrimSpace(s.PrincipalID); id != "" {
		return prefix + ":" + id
	}
	return prefix + ":0"
}
