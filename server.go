package main

import (
	"fmt"
	"maikani/handler"
	"net/http"

	"github.com/labstack/echo"
	"github.com/patrickmn/go-cache"
)

func main() {
	e := echo.New()
	client := &http.Client{}
	goCache := cache.New(cache.NoExpiration, cache.NoExpiration)
	h := &handler.Handler{
		Client:  client,
		GoCache: goCache,
	}
	fmt.Println(h)
	e.GET("/criticalSubjects", h.GetCriticalSubjects)
	e.Logger.Fatal(e.Start(":1234"))
}
