from fastapi import FastAPI, Request, Form
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from starlette.responses import RedirectResponse
from datetime import datetime
from typing import List
from pydantic import BaseModel

app = FastAPI()

# A simple in-memory message store
messages = []

class Message(BaseModel):
    username: str
    text: str
    timestamp: str

# Serve static files
# app.mount("/static", StaticFiles(directory="static"), name="static")

@app.get("/", response_class=HTMLResponse)
async def get_chat():
    """Render the chat page."""
    with open("templates/index.html") as file:
        return HTMLResponse(content=file.read())

@app.post("/send")
async def send_message(username: str = Form(...), text: str = Form(...)):
    """Receive a new message and store it."""
    timestamp = datetime.now().strftime('%H:%M')
    new_message = {"username": username, "text": text, "timestamp": timestamp}
    messages.append(new_message)
    return RedirectResponse(url="/messages", status_code=303)

@app.get("/messages", response_class=JSONResponse)
async def get_messages():
    """Return all messages as JSON."""
    return messages

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="127.0.0.1", port=8000)
