package main

import (
	"net/http"
	httphandlers "trading-dashboard/cmd/app-api/http-handlers"
	router "trading-dashboard/internal/httphandler/usecase/router"
	mw "trading-dashboard/internal/security/middleware"

	"go.uber.org/dig"
)

type Router struct {
	signIn     *httphandlers.SignIn
	signUp     *httphandlers.SignUp
	search     *httphandlers.Search
	marketData *httphandlers.MarketData
}
type routerParams struct {
	dig.In

	SignIn     *httphandlers.SignIn
	SignUp     *httphandlers.SignUp
	Search     *httphandlers.Search
	MarketData *httphandlers.MarketData
}

const basePath = "/api/v1"

type endpoint struct {
	path    string
	handler router.Handler
	secured bool
}

func newRouter(params routerParams) (*Router, error) {

	for _, e := range []endpoint{
		{
			path:    basePath + "/auth/register",
			handler: params.SignUp,
			secured: false,
		},
		{
			path:    basePath + "/auth/login",
			handler: params.SignIn,
			secured: false,
		},
		{
			path:    basePath + "/search",
			handler: params.Search,
			secured: true,
		},
		{
			path:    basePath + "/market-data",
			handler: params.MarketData,
			secured: true,
		},
	} {
		handleFunc := e.handler.Handle
		if e.secured {
			handleFunc = mw.JWTMiddleware(handleFunc)
		}

		http.HandleFunc(e.path, handleFunc)
	}

	r := &Router{
		signIn:     params.SignIn,
		signUp:     params.SignUp,
		search:     params.Search,
		marketData: params.MarketData,
	}

	return r, nil
}
