package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var venues = map[string]bool{"": true, "home": true, "away": true, "neutral": true}

// parseParams validates query parameters before any I/O happens.
func parseParams(q url.Values) (string, int, error) {
	venue := q.Get("venue")
	if !venues[venue] {
		return "", 0, errors.New("venue must be home, away or neutral")
	}
	s := q.Get("season")
	if s == "" {
		s = "2025"
	}
	season, err := strconv.Atoi(s)
	if err != nil || season < 2000 || season > 2100 {
		return "", 0, errors.New("season must be a year")
	}
	return venue, season, nil
}

type server struct {
	db  *sql.DB
	rdb *redis.Client
	ttl time.Duration
}

func (s *server) cached(ctx context.Context, key string, fn func() (any, error)) (any, error) {
	if b, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
		var v any
		if json.Unmarshal(b, &v) == nil {
			return v, nil
		}
	}
	v, err := fn()
	if err != nil {
		return nil, err
	}
	if b, e := json.Marshal(v); e == nil {
		s.rdb.Set(ctx, key, b, s.ttl)
	}
	return v, nil
}

func (s *server) handle(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		venue, season, err := parseParams(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		out, err := s.cached(ctx, "go:"+kind+":"+strconv.Itoa(season)+":"+venue, func() (any, error) {
			if kind == "matches" {
				return s.matches(ctx, venue, season)
			}
			return s.summary(ctx, venue, season)
		})
		if err != nil {
			log.Println("query failed:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	}
}

func (s *server) matches(ctx context.Context, venue string, season int) (any, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT match_date::text, opponent, venue, goals_for, goals_against
		FROM matches WHERE season=$1 AND ($2='' OR venue=$2) ORDER BY match_date`, season, venue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var d, o, v string
		var gf, ga int
		if err := rows.Scan(&d, &o, &v, &gf, &ga); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"date": d, "opponent": o, "venue": v, "gf": gf, "ga": ga})
	}
	return out, rows.Err()
}

func (s *server) summary(ctx context.Context, venue string, season int) (any, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT venue, COUNT(*), SUM((goals_for>goals_against)::int),
		SUM((goals_for=goals_against)::int), SUM(goals_for), SUM(goals_against), SUM((goals_against=0)::int)
		FROM matches WHERE season=$1 AND ($2='' OR venue=$2) GROUP BY venue`, season, venue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]any{}
	for rows.Next() {
		var v string
		var n, w, d, gf, ga, cs int
		if err := rows.Scan(&v, &n, &w, &d, &gf, &ga, &cs); err != nil {
			return nil, err
		}
		out[v] = map[string]any{"played": n, "wins": w, "draws": d, "losses": n - w - d, "gf": gf, "ga": ga,
			"clean_sheets": cs, "ppg": float64(3*w+d) / float64(n)}
	}
	return out, rows.Err()
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
		next.ServeHTTP(w, r)
	})
}

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(20)
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}
	s := &server{db: db, rdb: redis.NewClient(opt), ttl: 60 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("/api/v2/matches", s.handle("matches"))
	mux.HandleFunc("/api/v2/summary", s.handle("summary"))
	srv := &http.Server{Addr: ":8080", Handler: cors(mux), ReadHeaderTimeout: 5 * time.Second}
	log.Println("go-api listening on :8080")
	log.Fatal(srv.ListenAndServe())
}
