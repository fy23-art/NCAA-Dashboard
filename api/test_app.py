import os
os.environ.setdefault("DATABASE_URL", "postgresql://x:x@localhost/x")
from app import app

def test_health():
    assert app.test_client().get("/health").json == {"status": "ok"}

def test_rejects_bad_venue():
    assert app.test_client().get("/api/matches?venue=moon").status_code == 400

def test_rejects_bad_season():
    assert app.test_client().get("/api/summary?season=abc").status_code == 400
