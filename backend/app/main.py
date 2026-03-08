"""
Pi Terminal World Monitor – Backend API.

Runs on VPS; serves pre-aggregated panel data to the terminal client.
"""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(
    title="Pi Terminal World Monitor API",
    description="Backend for the Pi terminal dashboard; panel data only.",
    version="0.1.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
def health():
    """Health check for deployment and client connectivity."""
    return {"status": "ok", "service": "pi-terminal-world-monitor-backend"}
