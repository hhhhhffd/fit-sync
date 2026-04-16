import asyncio
import logging
from datetime import time
from aiogram import Bot, Dispatcher
from aiogram.filters import Command
from aiogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton, WebAppInfo
from aiogram.fsm.storage.memory import MemoryStorage

from config import BOT_TOKEN, WEBAPP_URL
from handlers import auth, profile, activity, challenges
from notifications import DailyReminderScheduler
from messages import msg, btn

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize bot and dispatcher with FSM storage
storage = MemoryStorage()
bot = Bot(token=BOT_TOKEN)
dp = Dispatcher(storage=storage)

# Initialize daily reminder scheduler (sends at 08:00 by default)
reminder_scheduler = DailyReminderScheduler(bot, reminder_time=time(8, 0))

# Register routers
dp.include_router(auth.router)
dp.include_router(profile.router)
dp.include_router(activity.router)
dp.include_router(challenges.router)

@dp.message(Command("menu", "help"))
async def cmd_menu(message: Message):
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=btn('open_webapp'), web_app=WebAppInfo(url=WEBAPP_URL))]
    ])
    await message.answer(msg('menu.header'), reply_markup=keyboard)

@dp.message(Command("webapp"))
async def cmd_webapp(message: Message):
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=btn('open_webapp'), web_app=WebAppInfo(url=WEBAPP_URL))]
    ])
    await message.answer(msg('menu.webapp_button'), reply_markup=keyboard)

async def main():
    logger.info("Starting bot...")
    reminder_scheduler.start()
    logger.info("Daily reminder scheduler started")

    try:
        await dp.start_polling(bot)
    finally:
        reminder_scheduler.stop()
        await bot.session.close()
        logger.info("Bot stopped")

if __name__ == "__main__":
    asyncio.run(main())
