package middleware
import (
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, util.Envelope{
					Code: string(util.CodeInternal), Message: "an unexpected error occurred",
					RequestID: util.RequestID(c),
				})
			}
		}()
		c.Next()
	}
}
