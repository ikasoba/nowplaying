package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strconv"

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
	Url    string
	Icon   string
	Title  string
	Status string
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

		return c.Render(http.StatusOK, "embed.html",
			`<a href="https://nowplaying.ikasoba.net/playing/`+user+`/url">
  <img src="https://nowplaying.ikasoba.net/playing/`+user+`" />
</a>`)
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

		icon, err := ToDataUrl(track.Images[len(track.Images)-1].Url)
		if err != nil {
			return err
		}

		header := c.Response().Header()
		header.Set("Cache-Control", "max-age="+strconv.Itoa(60*3))

		return c.Render(http.StatusOK, "playing.svg", Playing{
			Url:    track.Url,
			Icon:   icon,
			Title:  track.Name,
			Status: track.Artist.Name,
		})
	})

	addr := os.Getenv("HOST_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	e.Logger.Fatal(e.Start(addr))
}
