import csv, json, os
import psycopg2, redis
from flask import Flask, jsonify, request, abort

app = Flask(__name__)
VENUES = {"home", "away", "neutral"}
cache = redis.Redis.from_url(os.getenv("REDIS_URL", "redis://redis:6379/0"))
TTL = int(os.getenv("CACHE_TTL", "60"))

def db():
    return psycopg2.connect(os.environ["DATABASE_URL"])

def seed():
    with db() as c, c.cursor() as cur, open("/data/matches.csv") as f:
        cur.execute(open("/db/schema.sql").read())
        for r in csv.DictReader(f):
            cur.execute("""INSERT INTO matches (match_date,opponent,venue,goals_for,goals_against,season)
                VALUES (%(match_date)s,%(opponent)s,%(venue)s,%(goals_for)s,%(goals_against)s,%(season)s)
                ON CONFLICT DO NOTHING""", r)

def params():
    venue = request.args.get("venue")
    season = request.args.get("season", "2025")
    if venue is not None and venue not in VENUES:
        abort(400, "venue must be home, away or neutral")
    if not season.isdigit():
        abort(400, "season must be a year")
    return venue, int(season)

def cached(key, fn):
    hit = cache.get(key)
    if hit:
        return json.loads(hit)
    val = fn()
    cache.setex(key, TTL, json.dumps(val, default=str))
    return val

@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/api/matches")
def matches():
    venue, season = params()
    def q():
        with db() as c, c.cursor() as cur:
            cur.execute("""SELECT match_date,opponent,venue,goals_for,goals_against FROM matches
                WHERE season=%s AND (%s IS NULL OR venue=%s) ORDER BY match_date""", (season, venue, venue))
            return [dict(zip(("date","opponent","venue","gf","ga"), r)) for r in cur.fetchall()]
    return jsonify(cached(f"m:{season}:{venue}", q))

@app.get("/api/summary")
def summary():
    venue, season = params()
    def q():
        with db() as c, c.cursor() as cur:
            cur.execute("""SELECT venue, COUNT(*), SUM((goals_for>goals_against)::int),
                SUM((goals_for=goals_against)::int), SUM(goals_for), SUM(goals_against),
                SUM((goals_against=0)::int) FROM matches WHERE season=%s AND (%s IS NULL OR venue=%s)
                GROUP BY venue""", (season, venue, venue))
            out = {}
            for v, n, w, d, gf, ga, cs in cur.fetchall():
                out[v] = dict(played=n, wins=w, draws=d, losses=n-w-d, gf=gf, ga=ga,
                              clean_sheets=cs, ppg=round((3*w+d)/n, 2))
            return out
    return jsonify(cached(f"s:{season}:{venue}", q))

if __name__ == "__main__":
    seed()
    app.run(host="0.0.0.0", port=8000)
