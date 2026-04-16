import aiohttp
from datetime import datetime
from aiogram import Router, F
from aiogram.filters import Command
from aiogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton, CallbackQuery

from config import API_URL
from handlers.auth import get_token, is_authenticated
from messages import msg

router = Router()

@router.message(Command("activities"))
async def cmd_activities(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    args = message.text.split()
    if len(args) > 1 and args[1] in ['week', 'month', 'year', 'all']:
        await show_activities(message, args[1])
        return
    
    # Нет периода — показываем выбор
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [
            InlineKeyboardButton(text="📅 Неделя", callback_data="act_week"),
            InlineKeyboardButton(text="📆 Месяц", callback_data="act_month")
        ],
        [
            InlineKeyboardButton(text="🗓 Год", callback_data="act_year"),
            InlineKeyboardButton(text="📚 Всё", callback_data="act_all")
        ]
    ])
    await message.answer("📊 Выбери период для активностей:", reply_markup=keyboard)

@router.callback_query(F.data.startswith("act_"))
async def callback_activities(callback: CallbackQuery):
    period = callback.data.split("_")[1]
    await callback.answer()
    await show_activities(callback.message, period, callback.from_user.id)

async def show_activities(message: Message, period: str, user_id: int = None):
    if user_id is None:
        user_id = message.from_user.id if hasattr(message, 'from_user') and message.from_user else None
    
    if not user_id or not is_authenticated(user_id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(user_id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(
                f"{API_URL}/activities?period={period}",
                headers=headers
            ) as resp:
                if resp.status == 200:
                    activities = await resp.json()

                    if not activities:
                        await message.answer(msg('activities.empty', period=period))
                        return

                    text = msg('activities.header', period=period) + "\n\n"
                    for act in activities[:10]:
                        date = datetime.fromisoformat(act['activity_date'].replace('Z', '+00:00'))
                        text += (
                            f"• {act['activity_type'].upper()}\n"
                            f"  {act['duration']} мин | {act['distance']} км | {act['calories']} ккал\n"
                            f"  {date.strftime('%d.%m.%Y')}\n"
                        )
                        if act.get('notes'):
                            text += f"  📝 {act['notes']}\n"
                        text += "\n"

                    text += msg('activities.total', count=len(activities))
                    await message.answer(text)
                else:
                    await message.answer(msg('activities.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(Command("addactivity"))
async def cmd_add_activity(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    await message.answer(msg('activities.add_help'))
