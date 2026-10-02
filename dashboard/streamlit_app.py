import os
import pandas as pd
import requests
import streamlit as st

API = os.getenv("API_URL", "http://localhost:8000")
st.title("Rice Owls Women's Soccer: Home vs Away")
season = st.sidebar.selectbox("Season", [2024, 2025, 2026], index=1)
rows = requests.get(f"{API}/api/matches", params={"season": season}, timeout=5).json()
df = pd.DataFrame(rows)
df["result"] = (df.gf > df.ga).map({True: "W", False: "L"}).where(df.gf != df.ga, "D")
df["pts"] = df.result.map({"W": 3, "D": 1, "L": 0})
st.dataframe(df)
by = df.groupby("venue").agg(played=("gf", "size"), pts=("pts", "mean"), gf=("gf", "mean"), ga=("ga", "mean")).round(2)
st.bar_chart(by[["gf", "ga"]])
st.caption("pts = points per game by venue")
