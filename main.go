package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/damongolding/immich-kiosk/config"
	"github.com/damongolding/immich-kiosk/routes"
	"github.com/damongolding/immich-kiosk/utils"
)

// version current build version number
var version string

//go:embed public
var public embed.FS

func init() {
	routes.KioskVersion = version

	debugModeEnv := os.Getenv("KIOSK_DEBUG")
	debugMode, _ := strconv.ParseBool(debugModeEnv)

	if debugMode {
		log.SetLevel(log.DebugLevel)
		log.Debug("DEBUG mode on")
		zone, _ := time.Now().Zone()
		log.Debug("🕐", "current_time", time.Now().Format(time.Kitchen), "current_zone", zone)
	}

}

func main() {

	var baseConfig config.Config

	err := baseConfig.Load()
	if err != nil {
		log.Fatal("Failed to load config", "err", err)
	}

	fmt.Println(smallBanner)
	fmt.Print("Version ", version, "\n\n")

	e := echo.New()

	// hide echos default banner
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	if baseConfig.Kiosk.Password != "" {
		e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			Skipper: func(c echo.Context) bool {
				// skip auth for assets
				if strings.HasPrefix(c.Request().URL.String(), "/assets") {
					return true
				}
				return false
			},
			KeyLookup: "query:password",
			Validator: func(queryPassword string, c echo.Context) (bool, error) {
				return queryPassword == baseConfig.Kiosk.Password, nil
			},
			ErrorHandler: func(err error, c echo.Context) error {
				return c.String(http.StatusUnauthorized, "Unauthorized")
			},
		}))
	}
	if log.GetLevel() != log.DebugLevel {
		logger := log.New(os.Stdout)
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogURI:       true,
			LogStatus:    true,
			LogRequestID: true,
			LogRemoteIP:  true,
			LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
				requestColor := utils.StringToColor(values.RequestID)
				rColor := lipgloss.NewStyle().Foreground(lipgloss.Color(requestColor.Hex)).Render(values.RequestID)
				logger.WithPrefix(rColor).Info("status", values.Status, "URI", values.URI)
				return nil
			},
		}))
	}

	// CSS cache busting
	e.FileFS("/assets/css/style.*.css", "public/assets/css/style.css", public)

	// serve embdedd staic assets
	e.StaticFS("/assets", echo.MustSubFS(public, "public/assets"))

	e.GET("/", routes.Home(baseConfig))

	e.GET("/image", routes.NewImage(baseConfig))

	e.GET("/clock", routes.Clock(baseConfig))

	err = e.Start(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
