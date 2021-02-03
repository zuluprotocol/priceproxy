package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/priceproxy/config"
	"code.vegaprotocol.io/priceproxy/pricing"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// ErrorResponse is used when something went wrong.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Service is the HTTP service.
type Service struct {
	*httprouter.Router

	config config.Config
	server *http.Server
	pe     pricing.Engine
}

// PriceResponse gives the detail on one price.
type PriceResponse struct {
	Source            string  `json:"source"`
	Base              string  `json:"base"`
	Quote             string  `json:"quote"`
	Price             float64 `json:"price"`
	LastUpdatedReal   string  `json:"lastUpdatedReal"`
	LastUpdatedWander string  `json:"lastUpdatedWander"`
}

// PricesResponse gives details on multiple prices.
type PricesResponse struct {
	Prices []*PriceResponse `json:"prices"`
}

// NewService creates a new service instance (with optional mocks for test purposes).
func NewService(config config.Config) (s *Service, err error) {
	s = &Service{
		Router: httprouter.New(),
		config: config,
	}

	if err := s.initPricingEngine(); err != nil {
		return nil, fmt.Errorf("failed to initialise price engine: %s", err.Error())
	}

	s.addRoutes()
	s.server = s.getServer()

	return
}

func (s *Service) addRoutes() {
	s.GET("/prices", s.PricesGet)
	s.GET("/sources", s.SourcesGet)
	s.GET("/sources/:name", s.SourceGet)
	s.GET("/status", s.StatusGet)
}

func (s *Service) getServer() *http.Server {
	var handler http.Handler = s // cors.AllowAll().Handler(s)
	return &http.Server{
		Addr:           s.config.Server.Listen,
		WriteTimeout:   time.Second * 15,
		ReadTimeout:    time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20,
		Handler:        handler,
	}
}

// Start starts the HTTP server, and returns the server's exit error (if any).
func (s *Service) Start() error {
	log.WithFields(log.Fields{
		"listen": s.config.Server.Listen,
	}).Info("Listening")
	return s.server.ListenAndServe()
}

// Stop stops the HTTP service.
func (s *Service) Stop() {
	wait := 2 * time.Second
	log.WithFields(log.Fields{
		"listen": s.config.Server.Listen,
	}).Info("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Info("Server shutdown failed")
	}
}

func (s *Service) initPricingEngine() error {
	s.pe = pricing.NewEngine()
	for _, sourcecfg := range s.config.Sources {
		err := s.pe.AddSource(*sourcecfg)
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err.Error(),
				"name":        sourcecfg.Name,
				"sleepReal":   sourcecfg.SleepReal,
				"sleepWander": sourcecfg.SleepWander,
				"url":         sourcecfg.URL.String(),
			}).Fatal("Failed to add source")
		}
		log.WithFields(log.Fields{
			"name":        sourcecfg.Name,
			"sleepReal":   sourcecfg.SleepReal,
			"sleepWander": sourcecfg.SleepWander,
			"url":         sourcecfg.URL.String(),
		}).Info("Added source")
	}

	for _, pricecfg := range s.config.Prices {
		pi, err := s.pe.AddPrice(*pricecfg)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  err.Error(),
				"source": pricecfg.Source,
				"base":   pricecfg.Base,
				"quote":  pricecfg.Quote,
				"wander": pricecfg.Wander,
			}).Fatal("Failed to add price")
		}
		log.WithFields(log.Fields{
			"source": pricecfg.Source,
			"base":   pricecfg.Base,
			"quote":  pricecfg.Quote,
			"wander": pricecfg.Wander,
			"price":  pi.Price,
		}).Info("Added price")
	}

	return nil
}

// PricesGet gets information on all prices.
func (s *Service) PricesGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	source := r.URL.Query().Get("source")
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")
	var wanderPtr *bool
	wanderString := r.URL.Query().Get("wander")
	if wanderString != "" {
		wander, err := strconv.ParseBool(wanderString)
		if err != nil {
			writeError(w, fmt.Errorf("failed to parse wander as boolean"), http.StatusInternalServerError)
			return
		}
		wanderPtr = &wander
	}

	response := PricesResponse{
		Prices: make([]*PriceResponse, 0),
	}
	for k, v := range s.pe.GetPrices() {
		if (source == "" || source == k.Source) &&
			(base == "" || base == k.Base) &&
			(quote == "" || quote == k.Quote) &&
			(wanderPtr == nil || *wanderPtr == k.Wander) {

			response.Prices = append(response.Prices, &PriceResponse{
				Source:            k.Source,
				Base:              k.Base,
				Quote:             k.Quote,
				Price:             v.Price,
				LastUpdatedReal:   v.LastUpdatedReal.String(),
				LastUpdatedWander: v.LastUpdatedWander.String(),
			})
		}
	}
	writeSuccess(w, response, http.StatusOK)
}

// SourceGet gets information on one price.
func (s *Service) SourceGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	name := ps.ByName("name")

	source, err := s.pe.GetSource(name)
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	writeSuccess(w, source, http.StatusOK)
}

// SourcesGet gets information on all prices.
func (s *Service) SourcesGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sources, err := s.pe.GetSources()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	writeSuccess(w, sources, http.StatusOK)
}

// StatusGet says all is well.
func (s *Service) StatusGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	status := struct {
		Status bool
	}{
		Status: true,
	}
	writeSuccess(w, status, http.StatusOK)
}

func writeSuccess(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(data)
	_, _ = w.Write(buf)
}

func writeError(w http.ResponseWriter, e error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf, _ := json.Marshal(ErrorResponse{Error: e.Error()})
	_, _ = w.Write(buf)
}
