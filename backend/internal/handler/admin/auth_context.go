package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func requireAdminSubject(c *gin.Context) (middleware.AuthSubject, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return middleware.AuthSubject{}, false
	}
	if subject.IsSystem {
		return subject, true
	}
	if subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return middleware.AuthSubject{}, false
	}
	return subject, true
}

func requireHumanAdminSubject(c *gin.Context, message string) (middleware.AuthSubject, bool) {
	subject, ok := requireAdminSubject(c)
	if !ok {
		return middleware.AuthSubject{}, false
	}
	if subject.IsSystem || subject.UserID <= 0 {
		if message == "" {
			message = "Admin JWT required"
		}
		response.Forbidden(c, message)
		return middleware.AuthSubject{}, false
	}
	return subject, true
}

func adminActorUserID(subject middleware.AuthSubject) int64 {
	return subject.ActorUserID()
}
