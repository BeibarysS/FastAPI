from typing import Optional
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class Item(BaseModel):
    name: str
    price: float

@app.get("/")
def home():
    return {"message": "Сәлем, FastAPI!"}

@app.get("/items/{item_id}")
def get_item(item_id: int, q: Optional[str] = None):
    return {"id": item_id, "q": q}

@app.post("/items")
def create_item(item: Item):
    return {"saved": item}