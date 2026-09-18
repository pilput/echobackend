package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupBookmarkRoutes(api *echo.Group) {
	bookmarks := api.Group("/bookmarks", r.authMiddleware.Auth())
	{
		bookmarks.POST("/:post_id", r.bookmarkHandler.ToggleBookmark)
		bookmarks.GET("", r.bookmarkHandler.GetBookmarks)
		bookmarks.PATCH("/:bookmark_id", r.bookmarkHandler.UpdateBookmark)
		bookmarks.PATCH("/:bookmark_id/move", r.bookmarkHandler.MoveBookmark)
		bookmarks.POST("/folders", r.bookmarkHandler.CreateFolder)
		bookmarks.GET("/folders", r.bookmarkHandler.GetFolders)
		bookmarks.PATCH("/folders/:folder_id", r.bookmarkHandler.UpdateFolder)
		bookmarks.DELETE("/folders/:folder_id", r.bookmarkHandler.DeleteFolder)
	}
}
