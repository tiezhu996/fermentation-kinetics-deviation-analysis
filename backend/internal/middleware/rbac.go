package middleware
import (
	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
	"net/http"
)
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := ActorFromContext(c); !ok {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication context is missing"))
			return
		}
		c.Next()
	}
}
func RequireRoles(roles ...constants.Role) gin.HandlerFunc {
	allowed := make(map[constants.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		if _, ok := ActorFromContext(c); !ok {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication context is missing"))
			return
		}
		c.Next()
	}
}
