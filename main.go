package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"unicode/utf8"

	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/shkh/lastfm-go/lastfm"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type Playing struct {
	Url     string
	Icon    string
	Title   string
	Status  string
	Animate bool
}

type EmbedCode struct {
	LinkUrl  string
	ImageUrl string
}

func ToDataUrl(imgUrl string) (string, error) {
	resp, err := http.Get(imgUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)

	return "data:" + resp.Header.Get("Content-Type") + ";base64," + (base64.StdEncoding.EncodeToString(buf.Bytes())), nil
}

func main() {
	fm := lastfm.New(os.Getenv("LASTFM_KEY"), os.Getenv("LASTFM_SECRET"))
	e := echo.New()

	e.Renderer = &Template{
		templates: template.Must(template.ParseGlob("views/*")),
	}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		e.Logger.Error(err)
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.GET("/embed_code", func(c echo.Context) error {
		user := c.QueryParam("user")

		req := c.Request()

		var host string
		if req.Header.Get("X-Forwarded-Proto") == "https" {
			host = "https://"
		} else {
			host = "http://"
		}

		host += req.Host

		link, err := url.JoinPath(host, "/playing/", user, "/url")
		if err != nil {
			return err
		}

		imageUrl, err := url.JoinPath(host, "/playing/", user)
		if err != nil {
			return err
		}

		return c.Render(http.StatusOK, "embed.html", EmbedCode{
			LinkUrl:  link,
			ImageUrl: imageUrl,
		})
	})

	e.GET("/playing/:user/url", func(c echo.Context) error {
		res, err := fm.User.GetRecentTracks(lastfm.P{
			"user": c.Param("user"),
		})
		if err != nil {
			return err
		}

		track := res.Tracks[0]

		return c.Redirect(http.StatusFound, track.Url)
	})

	e.GET("/playing/:user", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "image/svg+xml")

		res, err := fm.User.GetRecentTracks(lastfm.P{
			"user": c.Param("user"),
		})
		if err != nil {
			return err
		}

		track := res.Tracks[0]

		icon, err := ToDataUrl(track.Images[0].Url)
		if err != nil {
			return err
		}

		header := c.Response().Header()
		header.Set("Cache-Control", "max-age="+strconv.Itoa(60*3))

		animate := false
		if utf8.RuneCountInString(track.Name) > 10 {
			animate = true
		}

		return c.Render(http.StatusOK, "playing.svg", Playing{
			Url:     track.Url,
			Icon:    icon,
			Title:   track.Name,
			Status:  track.Artist.Name,
			Animate: animate,
		})
	})

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = ":8080"
	}

	e.Logger.Fatal(e.Start(addr))
}
