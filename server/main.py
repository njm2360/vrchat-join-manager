import logging
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from db import close_db, init_db
from routers import analytics, events, groups, locations, players, worlds

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s"
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_db()
    yield
    await close_db()


app = FastAPI(lifespan=lifespan)

app.include_router(analytics.router)
app.include_router(events.router)
app.include_router(groups.router)
app.include_router(locations.router)
app.include_router(players.router)
app.include_router(worlds.router)

app.mount(
    "/",
    StaticFiles(directory=Path(__file__).parent / "static", html=True),
    name="static",
)
