package handler

import (
	"net/http"

	"github.com/patrickmn/go-cache"
)

//Handler containing needed resources (client and cache)
type Handler struct {
	Client  *http.Client
	GoCache *cache.Cache
}
