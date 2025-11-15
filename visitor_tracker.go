package traefik_visitor_middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Config struct {
	InfluxDBURL    string `json:"influxdb_url"`
	InfluxDBToken  string `json:"influxdb_token"`
	InfluxDBOrg    string `json:"influxdb_org"`
	InfluxDBBucket string `json:"influxdb_bucket"`
	HashSalt       string `json:"hash_salt"`
}

func CreateConfig() *Config {
	return &Config{
		InfluxDBURL:    "http://localhost:8086",
		InfluxDBToken:  "",
		InfluxDBOrg:    "",
		InfluxDBBucket: "visitors",
		HashSalt:       "default-salt",
	}
}

type VisitorTracker struct {
	next     http.Handler
	name     string
	config   *Config
	writeAPI api.WriteAPI
	client   influxdb2.Client
}

type VisitorData struct {
	IPHash    string    `json:"ip_hash"`
	Timestamp time.Time `json:"timestamp"`
	Domain    string    `json:"domain"`
	Path      string    `json:"path"`
	UserAgent string    `json:"user_agent"`
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.InfluxDBToken == "" {
		return nil, fmt.Errorf("influxdb_token is required")
	}
	if config.InfluxDBOrg == "" {
		return nil, fmt.Errorf("influxdb_org is required")
	}

	client := influxdb2.NewClient(config.InfluxDBURL, config.InfluxDBToken)
	writeAPI := client.WriteAPI(config.InfluxDBOrg, config.InfluxDBBucket)

	return &VisitorTracker{
		next:     next,
		name:     name,
		config:   config,
		writeAPI: writeAPI,
		client:   client,
	}, nil
}

func (vt *VisitorTracker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	vt.trackVisitor(req)
	vt.next.ServeHTTP(rw, req)
}

func (vt *VisitorTracker) trackVisitor(req *http.Request) {
	visitor := vt.extractVisitorData(req)
	vt.sendToInfluxDB(visitor)
}

func (vt *VisitorTracker) extractVisitorData(req *http.Request) VisitorData {
	clientIP := vt.getClientIP(req)
	ipHash := vt.hashIP(clientIP)

	return VisitorData{
		IPHash:    ipHash,
		Timestamp: time.Now(),
		Domain:    req.Host,
		Path:      req.URL.Path,
		UserAgent: req.Header.Get("User-Agent"),
	}
}

func (vt *VisitorTracker) getClientIP(req *http.Request) string {
	// Prüfe verschiedene Header für die echte Client-IP
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("Cf-Connecting-Ip"); ip != "" {
		return ip
	}
	return req.RemoteAddr
}

func (vt *VisitorTracker) hashIP(ip string) string {
	hasher := sha256.New()
	hasher.Write([]byte(ip + vt.config.HashSalt))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (vt *VisitorTracker) sendToInfluxDB(visitor VisitorData) {
	p := influxdb2.NewPoint("visitor_tracking",
		map[string]string{
			"domain":     visitor.Domain,
			"path":       visitor.Path,
			"ip_hash":    visitor.IPHash,
			"user_agent": visitor.UserAgent,
		},
		map[string]interface{}{
			"visit_count": 1,
		},
		visitor.Timestamp,
	)

	vt.writeAPI.WritePoint(p)
}
