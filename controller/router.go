// Pipe - A small and beautiful blogging platform written in golang.
// Copyright (C) 2017-2019, b3log.org & hacpai.com
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package controller

import (
	"errors"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/zou2699/mypipe/controller/console"
	"github.com/zou2699/mypipe/log"
	"github.com/zou2699/mypipe/model"
	"github.com/zou2699/mypipe/theme"
	"github.com/zou2699/mypipe/util"
)

// Logger
var logger = log.NewLogger(os.Stdout)

// MapRoutes returns a gin engine and binds controllers with request URLs.
func MapRoutes() *gin.Engine {
	ret := gin.New()
	ret.SetFuncMap(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("len(values) is " + strconv.Itoa(len(values)%2))
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"minus":    func(a, b int) int { return a - b },
		"mod":      func(a, b int) int { return a % b },
		"noescape": func(s string) template.HTML { return template.HTML(s) },
	})

	ret.Use(gin.Recovery())
	ret.Use(gin.Logger())
	store := sessions.NewCookieStore([]byte(model.Conf.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   model.Conf.SessionMaxAge,
		Secure:   strings.HasPrefix(model.Conf.Server, "https"),
		HttpOnly: true,
	})
	ret.Use(sessions.Sessions("pipe", store))
	ret.POST(util.PathUpload, uploadAction)
	ret.GET(util.PathPlatInfo, showPlatInfoAction)
	ret.GET(util.PathSitemap, outputSitemapAction)

	api := ret.Group(util.PathAPI)
	api.POST("/init", initAction)
	api.POST("/init/local", initLocalAction)
	api.POST("/register", registerAction)
	api.POST("/login", loginAction)
	api.POST("/logout", logoutAction)
	// 关闭文章推送
	//api.Any("/hp/*apis", util.HacPaiAPI())
	api.GET("/status", getStatusAction)
	// 关闭版本检查
	//api.GET("/check-version", console.CheckVersionAction)
	api.GET("/blogs/top", showTopBlogsAction)
	api.GET("/oauth/github/redirect", redirectGitHubLoginAction)
	api.GET("/oauth/github/callback", githubCallbackAction)

	consoleGroup := api.Group("/console")
	consoleGroup.Use(console.LoginCheck)

	if "dev" == model.Conf.RuntimeMode {
		consoleGroup.GET("/dev/articles/gen", console.GenArticlesAction)
	}

	consoleGroup.GET("/themes", console.GetThemesAction)
	consoleGroup.PUT("/themes/:id", console.UpdateThemeAction)
	consoleGroup.GET("/tags", console.GetTagsAction)
	consoleGroup.POST("/articles", console.AddArticleAction)
	consoleGroup.POST("/articles/batch-delete", console.RemoveArticlesAction)
	consoleGroup.GET("/articles", console.GetArticlesAction)
	consoleGroup.GET("/articles/:id", console.GetArticleAction)
	consoleGroup.DELETE("/articles/:id", console.RemoveArticleAction)
	consoleGroup.PUT("/articles/:id", console.UpdateArticleAction)
	consoleGroup.GET("/comments", console.GetCommentsAction)
	consoleGroup.POST("/comments/batch-delete", console.RemoveCommentsAction)
	consoleGroup.DELETE("/comments/:id", console.RemoveCommentAction)
	consoleGroup.GET("/categories", console.GetCategoriesAction)
	consoleGroup.POST("/categories", console.AddCategoryAction)
	consoleGroup.DELETE("/categories/:id", console.RemoveCategoryAction)
	consoleGroup.GET("/categories/:id", console.GetCategoryAction)
	consoleGroup.PUT("/categories/:id", console.UpdateCategoryAction)
	consoleGroup.GET("/navigations", console.GetNavigationsAction)
	consoleGroup.GET("/navigations/:id", console.GetNavigationAction)
	consoleGroup.PUT("/navigations/:id", console.UpdateNavigationAction)
	consoleGroup.POST("/navigations", console.AddNavigationAction)
	consoleGroup.DELETE("/navigations/:id", console.RemoveNavigationAction)
	consoleGroup.GET("/users", console.GetUsersAction)
	consoleGroup.POST("/users", console.AddUserAction)
	consoleGroup.GET("/thumbs", console.GetArticleThumbsAction)
	consoleGroup.POST("/markdown", console.MarkdownAction)
	consoleGroup.POST("/import/md", console.ImportMarkdownAction)
	consoleGroup.GET("/export/md", console.ExportMarkdownAction)
	// consoleGroup.POST("/blogs/switch/:id", console.BlogSwitchAction)

	consoleSettingsGroup := consoleGroup.Group("/settings")
	consoleSettingsGroup.GET("/basic", console.GetBasicSettingsAction)
	consoleSettingsGroup.PUT("/basic", console.UpdateBasicSettingsAction)
	consoleSettingsGroup.GET("/preference", console.GetPreferenceSettingsAction)
	consoleSettingsGroup.PUT("/preference", console.UpdatePreferenceSettingsAction)
	consoleSettingsGroup.GET("/sign", console.GetSignSettingsAction)
	consoleSettingsGroup.PUT("/sign", console.UpdateSignSettingsAction)
	consoleSettingsGroup.GET("/i18n", console.GetI18nSettingsAction)
	consoleSettingsGroup.PUT("/i18n", console.UpdateI18nSettingsAction)
	consoleSettingsGroup.GET("/feed", console.GetFeedSettingsAction)
	consoleSettingsGroup.PUT("/feed", console.UpdateFeedSettingsAction)
	consoleSettingsGroup.GET("/third-stat", console.GetThirdStatisticSettingsAction)
	consoleSettingsGroup.PUT("/third-stat", console.UpdateThirdStatisticSettingsAction)
	consoleSettingsGroup.GET("/ad", console.GetAdSettingsAction)
	consoleSettingsGroup.PUT("/ad", console.UpdateAdSettingsAction)
	consoleSettingsGroup.GET("/account", console.GetAccountAction)
	consoleSettingsGroup.PUT("/account", console.UpdateAccountAction)
	consoleSettingsGroup.PUT("/account/password", console.UpdatePasswordAction)

	ret.StaticFile(util.PathFavicon, staticPath("console/static/favicon.ico"))

	ret.Static(util.PathTheme+"/scss", staticPath("theme/scss"))
	ret.Static(util.PathTheme+"/js", staticPath("theme/js"))
	ret.Static(util.PathTheme+"/images", staticPath("theme/images"))
	ret.StaticFile("/sw.min.js", staticPath("theme/sw.min.js"))
	ret.StaticFile("/halt.html", staticPath("theme/halt.html"))

	for _, theme := range theme.Themes {
		themePath := staticPath("theme/x/" + theme)
		ret.Static("/theme/x/"+theme+"/css", themePath+"/css")
		ret.Static("/theme/x/"+theme+"/js", themePath+"/js")
		ret.Static("/theme/x/"+theme+"/images", themePath+"/images")
		ret.StaticFile("/theme/x/"+theme+"/thumbnail.jpg", themePath+"/thumbnail.jpg")
	}
	themeTemplates, err := filepath.Glob(staticPath("theme/x/*/*.html"))
	if nil != err {
		logger.Fatal("load theme templates failed: " + err.Error())
	}
	themeTemplates = append(themeTemplates, staticPath("theme/search/index.html"))
	commentTemplates, err := filepath.Glob(staticPath("theme/comment/*.html"))
	if nil != err {
		logger.Fatal("load comment templates failed: " + err.Error())
	}
	headTemplates, err := filepath.Glob(staticPath("theme/head/*.html"))
	if nil != err {
		logger.Fatal("load head templates failed: " + err.Error())
	}
	templates := append(themeTemplates, commentTemplates...)
	templates = append(templates, headTemplates...)
	ret.LoadHTMLFiles(templates...)
	themeGroup := ret.Group(util.PathBlogs + "/:username")
	themeGroup.Use(fillUser, pjax, resolveBlog)
	themeGroup.GET("", showArticlesAction)
	themeGroup.Any("/*path", routePath)

	adminPagesGroup := ret.Group(util.PathAdmin)
	adminPagesGroup.Use(fillUser)
	adminPagesGroup.GET("", console.ShowAdminPagesAction)
	adminPagesGroup.GET("/*path", console.ShowAdminPagesAction)

	indexGroup := ret.Group("")
	indexGroup.Use(fillUser)
	indexGroup.GET("", showIndexAction)
	indexGroup.GET(util.PathLogin, showLoginPageAction)
	indexGroup.GET(util.PathRegister, showRegisterPageAction)

	initGroup := ret.Group(util.PathInit)
	initGroup.Use(fillUser)
	initGroup.GET("", showInitPageAction)

	ret.Static(util.PathConsoleDist, staticPath("console/dist"))
	ret.StaticFile(util.PathChangelogs, staticPath("changelogs.html"))
	ret.StaticFile(util.PathRobots, staticPath("theme/robots.txt"))
	ret.NoRoute(func(c *gin.Context) {
		notFound(c)
	})

	return ret
}

func routePath(c *gin.Context) {
	path := c.Param("path")

	switch path {
	case util.PathActivities:
		showActivitiesAction(c)

		return
	case util.PathArchives:
		showArchivesAction(c)

		return
	case util.PathAuthors:
		showAuthorsAction(c)

		return
	case util.PathCategories:
		showCategoriesAction(c)

		return
	case util.PathTags:
		showTagsAction(c)

		return
	case util.PathComments:
		addCommentAction(c)

		return
	case util.PathAtom:
		outputAtomAction(c)

		return
	case util.PathRSS:
		outputRSSAction(c)

		return
	case util.PathUpload:
		uploadAction(c)

		return
	case util.PathFetchUpload:
		fetchUploadAction(c)

		return
	case util.PathSearch:
		searchAction(c)

		return
	case util.PathOpensearch:
		showOpensearchAction(c)

		return
	case util.PathAPIsSymComment:
		addSymCommentAction(c)

		return
	case util.PathAPIsSymArticle:
		if "POST" == c.Request.Method {
			addSymArticleAction(c)
		} else if "PUT" == c.Request.Method {
			updateSymArticleAction(c)
		}

		return
	}

	if strings.Contains(path, util.PathArchives+"/") {
		showArchiveArticlesAction(c)

		return
	}
	if strings.Contains(path, util.PathAuthors+"/") {
		showAuthorArticlesAction(c)

		return
	}
	if strings.Contains(path, util.PathCategories+"/") {
		showCategoryArticlesArticlesAction(c)

		return
	}
	if strings.Contains(path, util.PathTags+"/") {
		showTagArticlesAction(c)

		return
	}
	if strings.Contains(path, util.PathComments+"/") {
		if "DELETE" == c.Request.Method {
			commentID := strings.Split(c.Request.RequestURI, util.PathComments+"/")[1]
			c.Params = append(c.Params, gin.Param{Key: "id", Value: commentID})
			console.RemoveCommentAction(c)
		} else {
			getRepliesAction(c)
		}

		return
	}
	if "/" == path {
		showArticlesAction(c)

		return
	}

	logger.Tracef("can't handle path [" + path + "]")
	notFound(c)
}

func staticPath(relativePath string) string {
	return filepath.ToSlash(filepath.Join(model.Conf.StaticRoot, relativePath))
}
