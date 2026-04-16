import os
from pathlib import Path
from dotenv import load_dotenv

# Load .env from project root
env_path = Path(__file__).parent.parent / '.env'
load_dotenv(dotenv_path=env_path)

BOT_TOKEN = os.getenv('TELEGRAM_BOT_TOKEN', 'YOUR_BOT_TOKEN_HERE')
API_URL = os.getenv('API_URL', 'http://localhost:8080/api')
WEBAPP_URL = os.getenv('WEBAPP_URL', 'https://your-ngrok-url.ngrok.io')
