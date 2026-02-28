import os
import subprocess
from anthropic import Anthropic
from tavily import TavilyClient
from dotenv import load_dotenv

load_dotenv()

client = Anthropic()
tavily = TavilyClient(api_key=os.getenv("TAVILY_API_KEY"))

HISTORY_FILE = "history.txt"

def load_history() -> str:
    if not os.path.exists(HISTORY_FILE):
        return ""
    with open(HISTORY_FILE, "r") as f:
        return f.read().strip()

def save_to_history(entry: str):
    with open(HISTORY_FILE, "a") as f:
        f.write(entry + "\n")

def search_web(query: str) -> str:
    result = tavily.search(query=query, search_depth="advanced", max_results=3)
    return str(result)

tools = [
    {
        "name": "search_web",
        "description": "Search the web for literary passages or author information",
        "input_schema": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "The search query"
                }
            },
            "required": ["query"]
        }
    }
]

history = load_history()
history_note = f"\n\nAvoid these works you have already used:\n{history}" if history else ""

messages = [
    {
        "role": "user",
        "content": f"""Search the web for a short passage (around 100-200 words, not enough to introduce copyright concerns) from a great 
work of literary fiction. This should be like a quote.
*IMPORTANT* Return only the passage itself, followed by a single line with the author and work, then a short explanation on why this work was great.{history_note}

After the passage, on the very last line, write ONLY the author and title in this exact format so it can be logged:
LOGGED: Author - Title"""
    }
]

# Agentic loop
while True:
    response = client.messages.create(
        model="claude-sonnet-4-6",
        max_tokens=1024,
        tools=tools,
        messages=messages
    )

    if response.stop_reason == "end_turn":
        break

    if response.stop_reason == "tool_use":
        tool_use = next(b for b in response.content if b.type == "tool_use")
        tool_result = search_web(tool_use.input["query"])

        messages.append({"role": "assistant", "content": response.content})
        messages.append({
            "role": "user",
            "content": [
                {
                    "type": "tool_result",
                    "tool_use_id": tool_use.id,
                    "content": tool_result
                }
            ]
        })

final_text = next(b.text for b in response.content if hasattr(b, "text"))

# Extract and save the logged line, then remove it from display
lines = final_text.strip().splitlines()
logged_line = next((l for l in lines if l.startswith("LOGGED:")), None)
if logged_line:
    save_to_history(logged_line.replace("LOGGED:", "").strip())
    display_text = "\n".join(l for l in lines if not l.startswith("LOGGED:")).strip()
else:
    display_text = final_text

print(display_text)

subprocess.run([
    "notify-send",
    "--urgency=low",
    "--expire-time=30000",
    "Daily Passage",
    display_text
])
