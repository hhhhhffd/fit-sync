import aiohttp
from aiogram import Router, F
from aiogram.filters import Command, StateFilter
from aiogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton, WebAppInfo, CallbackQuery
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup

from config import API_URL, WEBAPP_URL
from messages import msg, btn, set_user_context

router = Router()

user_tokens = {}  # In production, use proper storage

# FSM States for registration
class RegistrationStates(StatesGroup):
    waiting_for_name = State()
    waiting_for_phone = State()
    waiting_for_physical_data = State()

@router.message(Command("start"))
async def cmd_start(message: Message, state: FSMContext):
    user_id = message.from_user.id
    username = message.from_user.username or f"user_{user_id}"
    first_name = message.from_user.first_name or ""
    last_name = message.from_user.last_name or ""

    if user_id in user_tokens:
        await message.answer(msg('auth.already_logged'))
        return

    # Try to login with Telegram ID
    async with aiohttp.ClientSession() as session:
        try:
            async with session.post(
                f"{API_URL}/telegram-auth",
                json={
                    "telegram_id": user_id,
                    "username": username,
                    "first_name": first_name,
                    "last_name": last_name
                }
            ) as resp:
                if resp.status == 200:
                    result = await resp.json()
                    token = result['token']
                    user = result['user']

                    user_tokens[user_id] = token
                    set_user_context(user)  # Auto-fill placeholders

                    is_new_user = user['total_wins'] == 0 and user['current_streak'] == 0

                    keyboard = InlineKeyboardMarkup(inline_keyboard=[
                        [InlineKeyboardButton(text=btn('open_webapp'), web_app=WebAppInfo(url=WEBAPP_URL))],
                        [InlineKeyboardButton(text=btn('show_menu'), callback_data="show_menu")]
                    ])

                    if is_new_user:
                        await message.answer(msg('auth.welcome_new'), reply_markup=keyboard)
                    else:
                        await message.answer(msg('auth.welcome_back'), reply_markup=keyboard)
                else:
                    error_text = await resp.text()
                    await message.answer(msg('errors.network') + f"\n\nDetails: {error_text}")
        except Exception as e:
            await message.answer(msg('errors.network') + f"\n\n{str(e)}")

@router.message(Command("register"))
async def cmd_register(message: Message, state: FSMContext):
    """Update profile information"""
    user_id = message.from_user.id

    if not is_authenticated(user_id):
        await message.answer(msg('auth.auth_required'))
        return

    await message.answer(
        "📝 Let's update your profile!\n\n"
        "Send me your name (or /skip to keep current):"
    )
    await state.set_state(RegistrationStates.waiting_for_name)

@router.message(StateFilter(RegistrationStates.waiting_for_name))
async def process_name(message: Message, state: FSMContext):
    if message.text == "/skip":
        name = None
    else:
        name = message.text.strip()

    await state.update_data(name=name)
    await message.answer(
        "📱 Send me your phone number (or /skip):"
    )
    await state.set_state(RegistrationStates.waiting_for_phone)

@router.message(StateFilter(RegistrationStates.waiting_for_phone))
async def process_phone(message: Message, state: FSMContext):
    if message.text == "/skip":
        phone = None
    else:
        phone = message.text.strip()

    await state.update_data(phone=phone)
    await message.answer(
        "📊 Send your physical data in format:\n"
        "age height(cm) weight(kg)\n\n"
        "Example: 25 175 70\n"
        "Or /skip"
    )
    await state.set_state(RegistrationStates.waiting_for_physical_data)

@router.message(StateFilter(RegistrationStates.waiting_for_physical_data))
async def process_physical_data(message: Message, state: FSMContext):
    age = height = weight = None

    if message.text != "/skip":
        try:
            parts = message.text.strip().split()
            if len(parts) == 3:
                age = int(parts[0])
                height = float(parts[1])
                weight = float(parts[2])
        except ValueError:
            await message.answer(msg('errors.invalid_input'))
            return

    # Get all data
    data = await state.get_data()

    # Update profile via API
    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

    profile_data = {}
    if data.get('name'):
        profile_data['name'] = data['name']
    if data.get('phone'):
        profile_data['phone'] = data['phone']
    if age:
        profile_data['age'] = age
    if height:
        profile_data['height'] = height
    if weight:
        profile_data['weight'] = weight

    async with aiohttp.ClientSession() as session:
        try:
            async with session.put(
                f"{API_URL}/profile",
                headers=headers,
                json=profile_data
            ) as resp:
                if resp.status == 200:
                    await message.answer(msg('profile.updated'))
                else:
                    await message.answer(msg('profile.error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

    await state.clear()

@router.message(Command("logout"))
async def cmd_logout(message: Message, state: FSMContext):
    user_id = message.from_user.id
    if user_id in user_tokens:
        del user_tokens[user_id]
        await state.clear()
        await message.answer(msg('auth.logged_out'))
    else:
        await message.answer(msg('auth.auth_required'))

@router.message(Command("link"))
async def cmd_link(message: Message):
    """Link Telegram account to existing website account"""
    user_id = message.from_user.id

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="🔗 Link Account", url=f"{WEBAPP_URL.replace('/webapp.html', '/login.html')}?telegram_id={user_id}")]
    ])

    await message.answer(
        "🔗 Link your Telegram to existing account:\n\n"
        "1. Click the button below\n"
        "2. Login with your credentials\n"
        "3. Your Telegram will be linked automatically",
        reply_markup=keyboard
    )

def get_token(user_id: int) -> str:
    return user_tokens.get(user_id)

def is_authenticated(user_id: int) -> bool:
    return user_id in user_tokens

# Handler for show_menu callback button
@router.callback_query(F.data == "show_menu")
async def callback_show_menu(callback: CallbackQuery):
    await callback.answer()
    await callback.message.answer(msg('menu.header'))
