package router

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/W1ndys/easy-qfnu-api-go/api/v1/admin"
	course_recommendation "github.com/W1ndys/easy-qfnu-api-go/api/v1/course_recommendation"
	"github.com/W1ndys/easy-qfnu-api-go/api/v1/questions"
	"github.com/W1ndys/easy-qfnu-api-go/api/v1/site"
	"github.com/W1ndys/easy-qfnu-api-go/api/v1/stats"
	zhjw "github.com/W1ndys/easy-qfnu-api-go/api/v1/zhjw"
	"github.com/W1ndys/easy-qfnu-api-go/common/response"
	"github.com/W1ndys/easy-qfnu-api-go/middleware"
	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由引擎
func InitRouter(webFS embed.FS) *gin.Engine {
	r := gin.Default()

	// 1. 注册中间件
	installMiddlewares(r)

	// 2. 注册 API 路由
	installAPIRoutes(r)

	// 3. 注册静态资源 (Web)
	installStaticRoutes(r, webFS)

	return r
}

func installMiddlewares(r *gin.Engine) {
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.Cors())
	r.Use(middleware.StatsCollector())
}

func installAPIRoutes(r *gin.Engine) {
	// 创建/api根路由组
	apiRoot := r.Group("/api")
	{
		// 健康检查接口
		apiRoot.GET("/health", func(c *gin.Context) {
			response.Success(c, "API is healthy")
		})
	}

	// 创建 v1 根组 (仅用于统一前缀 /api/v1)
	apiV1 := apiRoot.Group("/v1")

	// 【公开接口组】 (Public)
	// 特点：不挂载 AuthRequired 中间件
	{
		// apiV1.GET("/news", v1.GetNewsList)
		// apiV1.GET("/calendar", v1.GetCalendar)

		// 新生试题库搜索
		apiV1.GET("/questions/search", questions.Search)

		// 统计接口
		apiV1.GET("/stats/dashboard", stats.GetDashboard)
		apiV1.GET("/stats/trend", stats.GetTrend)

		// 站点公共接口
		siteGroup := apiV1.Group("/site")
		{
			siteGroup.POST("/verify", site.Verify)
			siteGroup.GET("/check-token", site.CheckToken)
			siteGroup.GET("/announcements", site.GetAnnouncements)
		}

		// 选课推荐公开接口
		courseRecGroup := apiV1.Group("/course-recommendation")
		{
			courseRecGroup.GET("/query", course_recommendation.Query)
			courseRecGroup.POST("/recommend", course_recommendation.Recommend)
		}

		// 教务系统登录接口（公开，不需要认证）
		apiV1.POST("/zhjw/login", zhjw.Login)
	}

	// 【受保护接口组】 (Protected)
	// zhjw 教务系统相关接口
	zhjwGroup := apiV1.Group("/zhjw")
	zhjwGroup.Use(middleware.AuthRequired())
	{
		// 成绩相关接口
		zhjwGroup.GET("/grade", zhjw.GetGradeList)
		// 教学计划/培养方案
		zhjwGroup.GET("/course-plan", zhjw.GetCoursePlan)
		// 考试安排相关接口
		zhjwGroup.GET("/exam", zhjw.GetExamSchedules)
		// 选课结果相关接口
		zhjwGroup.GET("/selection", zhjw.GetSelectionResults)
		// 课程表相关接口
		zhjwGroup.GET("/schedule", zhjw.GetClassSchedules)
	}

	// 管理后台接口
	adminGroup := apiV1.Group("/admin")
	{
		adminGroup.POST("/login", admin.Login)
		adminGroup.GET("/check-init", admin.CheckInit)
		adminGroup.POST("/init", admin.Init)

		// 需要认证的管理接口
		authAdmin := adminGroup.Group("")
		authAdmin.Use(middleware.AdminAuthRequired())
		{
			authAdmin.POST("/logout", admin.Logout)
			authAdmin.GET("/config", admin.GetConfig)
			authAdmin.POST("/config/update", admin.UpdateConfig)
			authAdmin.GET("/announcements", admin.GetAnnouncements)
			authAdmin.POST("/announcements", admin.CreateAnnouncement)
			authAdmin.POST("/announcements/:id/update", admin.UpdateAnnouncement)
			authAdmin.POST("/announcements/:id/delete", admin.DeleteAnnouncement)

			// 选课推荐管理
			authAdmin.GET("/course-recommendations", course_recommendation.GetAll)
			authAdmin.POST("/course-recommendations/review", course_recommendation.Review)
			authAdmin.POST("/course-recommendations/update", course_recommendation.Update)
			authAdmin.POST("/course-recommendations/delete", course_recommendation.Delete)
		}
	}
}

func installStaticRoutes(r *gin.Engine, webFS embed.FS) {
	// 1. 加载 HTML 模板
	r.HTMLRender = loadTemplates(webFS)

	// 2. 注册静态资源路由
	staticFiles, _ := fs.Sub(webFS, "web/static")
	r.StaticFS("/static", http.FS(staticFiles))

	// 3. 注册页面路由
	// 公开页面
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", nil)
	})
	r.GET("/access", func(c *gin.Context) {
		c.HTML(http.StatusOK, "access.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	// 管理后台页面
	r.GET("/admin/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/login.html", nil)
	})
	r.GET("/admin/init", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/init.html", nil)
	})
	adminPages := r.Group("/admin")
	adminPages.Use(middleware.AdminAuthRequired())
	{
		adminPages.GET("", func(c *gin.Context) {
			c.HTML(http.StatusOK, "admin/index.html", nil)
		})
		adminPages.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "admin/index.html", nil)
		})
	}

	// 受保护页面
	protected := r.Group("")
	protected.Use(middleware.SiteAccessRequired())
	{
		protected.GET("/grade", func(c *gin.Context) {
			c.HTML(http.StatusOK, "grade.html", nil)
		})
		protected.GET("/schedule", func(c *gin.Context) {
			c.HTML(http.StatusOK, "schedule.html", nil)
		})
		protected.GET("/course-plan", func(c *gin.Context) {
			c.HTML(http.StatusOK, "course-plan.html", nil)
		})
		protected.GET("/exam", func(c *gin.Context) {
			c.HTML(http.StatusOK, "exam.html", nil)
		})
		protected.GET("/selection", func(c *gin.Context) {
			c.HTML(http.StatusOK, "selection.html", nil)
		})
		protected.GET("/questions", func(c *gin.Context) {
			c.HTML(http.StatusOK, "questions.html", nil)
		})
		protected.GET("/course-recommendation", func(c *gin.Context) {
			c.HTML(http.StatusOK, "course-recommendation.html", nil)
		})
	}
}
