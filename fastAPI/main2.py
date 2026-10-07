from typing import Optional, List
from fastapi import FastAPI, HTTPException, status
from pydantic import BaseModel, Field

app = FastAPI(title="Items API")

# --- Models ---
class ItemCreate(BaseModel):
    name: str = Field(min_length=1, max_length=50)
    price: float = Field(gt=0)
    description: Optional[str] = None

class Item(ItemCreate):
    id: int

# --- Fake database (in memory) ---
db: dict[int, Item] = {}
next_id = 1

# --- Endpoints ---
@app.post("/items", response_model=Item, status_code=status.HTTP_201_CREATED)
def create_item(data: ItemCreate):
    global next_id
    item = Item(id=next_id, **data.dict())
    db[next_id] = item
    next_id += 1
    return item

@app.get("/items", response_model=List[Item])
def list_items(skip: int = 0, limit: int = 10):
    return list(db.values())[skip : skip + limit]

@app.get("/items/{item_id}", response_model=Item)
def get_item(item_id: int):
    if item_id not in db:
        raise HTTPException(status_code=404, detail="Item not found")
    return db[item_id]

@app.put("/items/{item_id}", response_model=Item)
def update_item(item_id: int, data: ItemCreate):
    if item_id not in db:
        raise HTTPException(status_code=404, detail="Item not found")
    db[item_id] = Item(id=item_id, **data.dict())
    return db[item_id]

@app.delete("/items/{item_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_item(item_id: int):
    if item_id not in db:
        raise HTTPException(status_code=404, detail="Item not found")
    del db[item_id]