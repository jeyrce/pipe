// Pipe - A small and beautiful blogging platform written in golang.
// Copyright (C) 2017, b3log.org
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

	"github.com/b3log/pipe/controller/console"
	"github.com/b3log/pipe/log"
	"github.com/b3log/pipe/theme"
	"github.com/b3log/pipe/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
		"minus": func(a, b int) int {
			return a - b
		},
	})

	ret.Use(gin.Recovery())

	store := sessions.NewCookieStore([]byte(util.Conf.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   util.Conf.SessionMaxAge,
		Secure:   strings.HasPrefix(util.Conf.Server, "https"),
		HttpOnly: true,
	})
	ret.Use(sessions.Sessions("pipe", store))

	api := ret.Group(util.PathAPI)
	api.POST("/init", initAction)
	api.POST("/logout", logoutAction)
	api.Any("/hp/*apis", util.HacPaiAPI())
	api.GET("/status", getStatusAction)
	api.GET("/check-version", console.CheckVersion)

	consoleGroup := api.Group("/console")
	consoleGroup.Use(console.LoginCheck)

	if "dev" == util.Conf.RuntimeMode {
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

	consoleGroup.POST("/blogs/switch/:id", console.BlogSwitchAction)

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

	ret.StaticFile(util.PathFavicon, "console/static/favicon.ico")

	ret.Static(util.PathTheme+"/css", "theme/css")
	ret.Static(util.PathTheme+"/js", "theme/js")

	for _, theme := range theme.Themes {
		themePath := "theme/x/" + theme
		ret.Static("/"+themePath+"/css", themePath+"/css")
		ret.Static("/"+themePath+"/js", themePath+"/js")
		ret.Static("/"+themePath+"/images", themePath+"/images")
	}
	themeTemplates, err := filepath.Glob("theme/x/*/*.html")
	if nil != err {
		logger.Fatal("load theme templates failed: " + err.Error())
	}
	commentTemplates, err := filepath.Glob("theme/comment/*.html")
	if nil != err {
		logger.Fatal("load comment templates failed: " + err.Error())
	}
	templates := append(themeTemplates, commentTemplates...)
	ret.LoadHTMLFiles(templates...)
	themeGroup := ret.Group(util.PathBlogs + "/:username")
	themeGroup.Use(fillUser, resolveBlog)
	themeGroup.GET("", showArticlesAction)
	themeGroup.Any("/*path", routePath)

	adminPagesGroup := ret.Group(util.PathAdmin)
	adminPagesGroup.Use(fillUser)
	adminPagesGroup.GET("/*path", console.ShowAdminPagesAction)

	indexGroup := ret.Group("")
	indexGroup.Use(fillUser)
	indexGroup.GET("", showIndexAction)

	initGroup := ret.Group(util.PathInit)
	initGroup.Use(fillUser)
	initGroup.GET("", showInitPageAction)

	searchGroup := ret.Group(util.PathSearch)
	searchGroup.Use(fillUser)
	searchGroup.GET("", showSearchPageAction)

	ret.Static(util.PathAssets, "./console/dist")

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
		getRepliesAction(c)

		return
	}

	logger.Infof("can't handle path [" + path + "]")
}
