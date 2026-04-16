import aiohttp
from aiogram import Router
from aiogram.filters import Command
from aiogram.types import Message

from config import API_URL
from handlers.auth import get_token, is_authenticated
from messages import msg, set_user_context

router = Router()

@router.message(Command("profile"))
async def cmd_profile(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/profile", headers=headers) as resp:
                if resp.status == 200:
                    user = await resp.json()
                    set_user_context(user)
                    
                    text = msg('profile.full',
                        name=user.get('name') or 'Не указано',
                        height=user.get('height') or 'Не указано',
                        weight=user.get('weight') or 'Не указано',
                        wins=user['total_wins'],
                        streak=user['current_streak'],
                        best_streak=user['best_streak']
                    )
                    await message.answer(text)
                else:
                    await message.answer(msg('profile.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(Command("stats"))
async def cmd_stats(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/profile", headers=headers) as resp:
                if resp.status == 200:
                    user = await resp.json()
                    set_user_context(user)
                    await message.answer(msg('profile.stats'))
                else:
                    await message.answer(msg('profile.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))
