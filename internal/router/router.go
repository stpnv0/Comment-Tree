// Package router sets up HTTP routing for the application.
package router

import (
	"net/http"

	"github.com/stpnv0/CommentTree/internal/handler"
	"github.com/wb-go/wbf/ginext"
)

func InitRouter(
	ginMode string,
	commentHandler *handler.CommentHandler,
	mw ...ginext.HandlerFunc,
) *ginext.Engine {
	r := ginext.New(ginMode)
	r.Use(ginext.Recovery())
	r.Use(mw...)

	r.POST("/comments", commentHandler.Create)
	r.GET("/comments", commentHandler.GetTree)
	r.DELETE("/comments/:id", commentHandler.Delete)
	r.GET("/comments/search", commentHandler.Search)

	r.GET("/health", func(c *ginext.Context) {
		c.JSON(http.StatusOK, ginext.H{"status": "ok"})
	})

	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")

	r.GET("/", func(c *ginext.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	return r
}
