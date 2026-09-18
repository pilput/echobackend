package routes

import "github.com/labstack/echo/v5"

func (r *Routes) setupReportRoutes(api *echo.Group) {
	reports := api.Group("/reports", r.authMiddleware.Auth(), r.authMiddleware.AuthAdmin())
	{
		reports.GET("/overview", r.reportHandler.GetOverview)
		reports.GET("/users", r.reportHandler.GetUsers)
		reports.GET("/posts", r.reportHandler.GetPosts)
		reports.GET("/engagement", r.reportHandler.GetEngagement)
	}
}
