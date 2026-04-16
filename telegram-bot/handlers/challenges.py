import aiohttp
from datetime import datetime, timedelta
from aiogram import Router, F
from aiogram.filters import Command, StateFilter
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup

from config import API_URL
from handlers.auth import get_token, is_authenticated
from messages import msg, btn

router = Router()

# FSM States for challenge creation wizard
class CreateChallengeStates(StatesGroup):
    waiting_for_type = State()
    waiting_for_title = State()
    waiting_for_description = State()
    waiting_for_goal_value = State()
    waiting_for_max_participants = State()
    waiting_for_duration = State()
    confirm = State()

# FSM States for progress tracking
class AddProgressStates(StatesGroup):
    waiting_for_challenge = State()
    waiting_for_value = State()
    waiting_for_photo = State()

# FSM States for waiting ID input
class WaitingChallengeIdStates(StatesGroup):
    waiting_for_challenge_id = State()
    waiting_for_join_code = State()
    waiting_for_progress_id = State()
    waiting_for_leaderboard_type = State()
    waiting_for_activity_period = State()

def render_progress_bar(current: int, goal: int, length: int = 10) -> str:
    if goal == 0:
        return "[" + "░" * length + "]"
    filled = int((current / goal) * length)
    filled = min(filled, length)
    empty = length - filled
    return "[" + "▓" * filled + "░" * empty + "]"

@router.message(Command("challenges"))
async def cmd_challenges(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/challenges", headers=headers) as resp:
                if resp.status == 200:
                    challenges = await resp.json()

                    if not challenges:
                        await message.answer(msg('challenges.list_empty'))
                        return

                    text = msg('challenges.list_header') + "\n\n"
                    for ch in challenges[:10]:
                        status_emoji = {'pending': '⏳', 'active': '🔥', 'completed': '✅'}.get(ch['status'], '❓')
                        type_emoji = '📊' if ch.get('type') == 'accumulative' else '📅'
                        type_name = 'Накопит.' if ch.get('type') == 'accumulative' else 'Последов.'

                        text += (
                            f"{status_emoji} {ch['title']}\n"
                            f"  {type_emoji} {type_name} | 🎯 {ch.get('goal_value', 0)}\n"
                            f"  ID: {ch['id']}\n\n"
                        )

                    text += msg('challenges.list_footer')
                    await message.answer(text)
                else:
                    await message.answer(msg('challenges.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(Command("challenge"))
async def cmd_challenge_detail(message: Message, state: FSMContext):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    args = message.text.split()
    if len(args) < 2:
        # Нет ID — показываем список и просим ввести
        token = get_token(message.from_user.id)
        headers = {"Authorization": f"Bearer {token}"}
        
        async with aiohttp.ClientSession() as session:
            try:
                async with session.get(f"{API_URL}/challenges", headers=headers) as resp:
                    if resp.status == 200:
                        challenges = await resp.json()
                        if challenges:
                            text = "📋 *Твои челленджи:*\n\n"
                            for ch in challenges[:10]:
                                status_emoji = {'pending': '⏳', 'active': '🔥', 'completed': '✅'}.get(ch['status'], '❓')
                                text += f"{status_emoji} `{ch['id']}` — {ch['title']}\n"
                            text += "\n✏️ Введи ID челленджа:"
                            await message.answer(text, parse_mode="Markdown")
                        else:
                            await message.answer("📋 У тебя пока нет челленджей.\n\nСоздай: /createchallenge\nПрисоединиться: /join <код>")
                            return
            except:
                await message.answer("✏️ Введи ID челленджа:")
        
        await state.set_state(WaitingChallengeIdStates.waiting_for_challenge_id)
        return

    await show_challenge_detail(message, args[1])

@router.message(StateFilter(WaitingChallengeIdStates.waiting_for_challenge_id))
async def process_challenge_id_input(message: Message, state: FSMContext):
    await state.clear()
    challenge_id = message.text.strip()
    await show_challenge_detail(message, challenge_id)

async def show_challenge_detail(message: Message, challenge_id: str):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/challenges/{challenge_id}", headers=headers) as resp:
                if resp.status != 200:
                    await message.answer(msg('challenges.not_found'))
                    return
                ch = await resp.json()

            async with session.get(f"{API_URL}/challenges/{challenge_id}/progress", headers=headers) as resp:
                progress_data = await resp.json() if resp.status == 200 else []

            start = datetime.fromisoformat(ch['start_date'].replace('Z', '+00:00'))
            end = datetime.fromisoformat(ch['end_date'].replace('Z', '+00:00'))

            type_emoji = '📊' if ch.get('type') == 'accumulative' else '📅'
            type_name = 'Накопительный' if ch.get('type') == 'accumulative' else 'Последовательный'
            goal = ch.get('goal_value', 0)

            participants_count = len(ch.get('participants', []))
            max_participants = ch.get('max_participants')
            participants_text = f"{participants_count}/{max_participants}" if max_participants else f"{participants_count} (безлимит)"

            text = (
                f"🏆 {ch['title']}\n\n"
                f"📄 {ch.get('description') or 'Без описания'}\n"
                f"{type_emoji} Тип: {type_name}\n"
                f"🎯 Цель: {goal}\n"
                f"👥 Участников: {participants_text}\n"
                f"📊 Статус: {ch['status']}\n"
                f"📅 {start.strftime('%d.%m.%Y')} - {end.strftime('%d.%m.%Y')}\n"
            )

            if ch.get('invite_code'):
                text += f"\n📨 Код: {ch['invite_code']}\n"
                text += f"Присоединиться: /join {ch['invite_code']}\n"

            if progress_data:
                text += "\n👥 Прогресс:\n"
                for p in progress_data:
                    participant = next((u for u in ch.get('participants', []) if u['id'] == p['user_id']), None)
                    name = participant.get('name') if participant else f"User #{p['user_id']}"
                    progress = p['current_progress']
                    bar = render_progress_bar(progress, goal)
                    text += f"  {name}: {bar} {progress}/{goal}\n"

            if ch.get('winner_id'):
                text += f"\n🎉 Победитель: User #{ch['winner_id']}"

            text += f"\n\n💪 Добавить прогресс: /progress {challenge_id}"
            await message.answer(text)
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(Command("achievements"))
async def cmd_achievements(message: Message):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/achievements", headers=headers) as resp:
                if resp.status == 200:
                    achievements = await resp.json()

                    if not achievements:
                        await message.answer(msg('achievements.list_empty'))
                        return

                    text = msg('achievements.list_header') + "\n\n"
                    for ach in achievements:
                        icon = ach['achievement'].get('icon', '🏅')
                        name = ach['achievement']['name']
                        desc = ach['achievement']['description']
                        text += f"{icon} {name}\n  {desc}\n\n"

                    await message.answer(text)
                else:
                    await message.answer(msg('achievements.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(Command("leaderboard"))
async def cmd_leaderboard(message: Message, state: FSMContext):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    args = message.text.split()
    if len(args) > 1:
        lb_type = args[1]
        if lb_type in ['wins', 'streak']:
            await show_leaderboard(message, lb_type)
            return
    
    # Нет типа — показываем выбор
    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="🏆 По победам", callback_data="lb_wins")],
        [InlineKeyboardButton(text="🔥 По сериям", callback_data="lb_streak")]
    ])
    await message.answer("📊 Выбери тип лидерборда:", reply_markup=keyboard)

@router.callback_query(F.data.startswith("lb_"))
async def callback_leaderboard(callback: CallbackQuery):
    lb_type = callback.data.split("_")[1]
    await callback.answer()
    await show_leaderboard(callback.message, lb_type, callback.from_user.id)

async def show_leaderboard(message: Message, lb_type: str, user_id: int = None):
    if user_id is None:
        user_id = message.from_user.id if hasattr(message, 'from_user') and message.from_user else None
    
    if not user_id or not is_authenticated(user_id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(user_id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/leaderboard?type={lb_type}", headers=headers) as resp:
                if resp.status == 200:
                    leaderboard = await resp.json()

                    if not leaderboard:
                        await message.answer(msg('leaderboard.empty'))
                        return

                    type_emoji = {'wins': '🏆', 'streak': '🔥'}.get(lb_type, '📊')
                    type_name = 'ПОБЕДЫ' if lb_type == 'wins' else 'СЕРИИ'

                    text = msg('leaderboard.header', emoji=type_emoji, type=type_name) + "\n\n"

                    for entry in leaderboard[:10]:
                        rank = entry['rank']
                        name = entry.get('name') or f"User #{entry['id']}"
                        value = entry['total_wins'] if lb_type == 'wins' else entry['best_streak']

                        medal = ""
                        if rank == 1: medal = "🥇 "
                        elif rank == 2: medal = "🥈 "
                        elif rank == 3: medal = "🥉 "

                        text += f"{medal}{rank}. {name}: {value}\n"

                    await message.answer(text)
                else:
                    await message.answer(msg('leaderboard.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

# ===== CHALLENGE CREATION WIZARD =====

@router.message(Command("createchallenge"))
async def cmd_create_challenge_start(message: Message, state: FSMContext):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=btn('type_accumulative'), callback_data="challenge_type_accumulative")],
        [InlineKeyboardButton(text=btn('type_consistency'), callback_data="challenge_type_consistency")],
        [InlineKeyboardButton(text=btn('cancel'), callback_data="challenge_cancel")]
    ])

    await message.answer(msg('challenges.create_start'), reply_markup=keyboard, parse_mode="Markdown")
    await state.set_state(CreateChallengeStates.waiting_for_type)

@router.callback_query(F.data.startswith("challenge_type_"), StateFilter(CreateChallengeStates.waiting_for_type))
async def process_challenge_type(callback: CallbackQuery, state: FSMContext):
    challenge_type = callback.data.split("_")[-1]

    if challenge_type not in ["accumulative", "consistency"]:
        await callback.answer("Неверный тип")
        return

    await state.update_data(type=challenge_type)
    await callback.answer()
    
    type_name = 'Накопительный' if challenge_type == 'accumulative' else 'Последовательный'
    await callback.message.edit_text(msg('challenges.type_selected', type_name=type_name))
    await callback.message.answer(msg('challenges.ask_title'))
    await state.set_state(CreateChallengeStates.waiting_for_title)

@router.message(StateFilter(CreateChallengeStates.waiting_for_title))
async def process_challenge_title(message: Message, state: FSMContext):
    title = message.text.strip()

    if len(title) < 3:
        await message.answer(msg('challenges.title_too_short'))
        return

    await state.update_data(title=title)
    await message.answer(msg('challenges.ask_description', title=title))
    await state.set_state(CreateChallengeStates.waiting_for_description)

@router.message(StateFilter(CreateChallengeStates.waiting_for_description))
async def process_challenge_description(message: Message, state: FSMContext):
    description = "" if message.text == "/skip" else message.text.strip()
    await state.update_data(description=description)

    data = await state.get_data()
    if data['type'] == "accumulative":
        await message.answer(msg('challenges.ask_goal_accumulative'))
    else:
        await message.answer(msg('challenges.ask_goal_consistency'))

    await state.set_state(CreateChallengeStates.waiting_for_goal_value)

@router.message(StateFilter(CreateChallengeStates.waiting_for_goal_value))
async def process_challenge_goal(message: Message, state: FSMContext):
    try:
        goal_value = int(message.text.strip())
        if goal_value <= 0:
            raise ValueError
    except ValueError:
        await message.answer(msg('challenges.invalid_goal'))
        return

    await state.update_data(goal_value=goal_value)
    await message.answer(msg('challenges.ask_max_participants'))
    await state.set_state(CreateChallengeStates.waiting_for_max_participants)

@router.message(StateFilter(CreateChallengeStates.waiting_for_max_participants))
async def process_max_participants(message: Message, state: FSMContext):
    max_participants = None

    if message.text.strip() != "/skip":
        try:
            max_participants = int(message.text.strip())
            if max_participants <= 0:
                await message.answer(msg('challenges.invalid_max_participants'))
                return
        except ValueError:
            await message.answer(msg('challenges.invalid_max_participants'))
            return

    await state.update_data(max_participants=max_participants)
    await message.answer(msg('challenges.ask_duration'))
    await state.set_state(CreateChallengeStates.waiting_for_duration)

@router.message(StateFilter(CreateChallengeStates.waiting_for_duration))
async def process_challenge_duration(message: Message, state: FSMContext):
    try:
        duration = int(message.text.strip())
        if duration <= 0:
            raise ValueError
    except ValueError:
        await message.answer(msg('challenges.invalid_duration'))
        return

    data = await state.get_data()
    start_date = datetime.utcnow()
    end_date = start_date + timedelta(days=duration)

    await state.update_data(
        start_date=start_date.strftime('%Y-%m-%dT%H:%M:%SZ'),
        end_date=end_date.strftime('%Y-%m-%dT%H:%M:%SZ'),
        duration=duration
    )

    type_name = 'Накопительный' if data['type'] == 'accumulative' else 'Последовательный'
    max_p = data.get('max_participants')
    max_p_text = f"{max_p}" if max_p else "Безлимит"

    text = msg('challenges.confirm_summary',
        title=data['title'],
        description=data.get('description') or 'Нет',
        type_name=type_name,
        goal=data['goal_value'],
        max_participants=max_p_text,
        duration=duration,
        start_date=start_date.strftime('%d.%m.%Y'),
        end_date=end_date.strftime('%d.%m.%Y')
    )

    keyboard = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=btn('confirm'), callback_data="challenge_confirm")],
        [InlineKeyboardButton(text=btn('cancel'), callback_data="challenge_cancel")]
    ])

    await message.answer(text, reply_markup=keyboard)
    await state.set_state(CreateChallengeStates.confirm)

@router.callback_query(F.data == "challenge_confirm", StateFilter(CreateChallengeStates.confirm))
async def confirm_challenge_creation(callback: CallbackQuery, state: FSMContext):
    data = await state.get_data()
    token = get_token(callback.from_user.id)
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

    challenge_data = {
        "title": data['title'],
        "description": data.get('description', ''),
        "type": data['type'],
        "goal_value": data['goal_value'],
        "start_date": data['start_date'],
        "end_date": data['end_date']
    }

    if data.get('max_participants') is not None:
        challenge_data["max_participants"] = data['max_participants']

    async with aiohttp.ClientSession() as session:
        try:
            async with session.post(f"{API_URL}/challenges", headers=headers, json=challenge_data) as resp:
                if resp.status == 200:
                    challenge = await resp.json()
                    await callback.answer("Создано!")
                    await callback.message.edit_text(
                        msg('challenges.created',
                            challenge_title=challenge['title'],
                            invite_code=challenge['invite_code'],
                            challenge_id=challenge['id']
                        ),
                        parse_mode="Markdown"
                    )
                else:
                    error = await resp.text()
                    await callback.message.edit_text(msg('challenges.creation_error', error=error))
        except Exception as e:
            await callback.message.edit_text(msg('errors.generic'))

    await state.clear()

@router.callback_query(F.data == "challenge_cancel")
async def cancel_challenge_creation(callback: CallbackQuery, state: FSMContext):
    await state.clear()
    await callback.answer("Отменено")
    await callback.message.edit_text(msg('challenges.creation_cancelled'))

# ===== JOIN CHALLENGE =====

@router.message(Command("join"))
async def cmd_join_challenge(message: Message, state: FSMContext):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    args = message.text.split()
    if len(args) < 2:
        await message.answer("📨 Введи код приглашения для присоединения к челленджу:")
        await state.set_state(WaitingChallengeIdStates.waiting_for_join_code)
        return

    await join_challenge_by_code(message, args[1])

@router.message(StateFilter(WaitingChallengeIdStates.waiting_for_join_code))
async def process_join_code_input(message: Message, state: FSMContext):
    await state.clear()
    invite_code = message.text.strip()
    await join_challenge_by_code(message, invite_code)

async def join_challenge_by_code(message: Message, invite_code: str):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.post(f"{API_URL}/challenges/join/{invite_code}", headers=headers) as resp:
                if resp.status == 200:
                    await message.answer(msg('challenges.joined'))
                else:
                    error = await resp.text()
                    await message.answer(msg('challenges.join_error', error=error))
        except Exception as e:
            await message.answer(msg('errors.generic'))

# ===== PROGRESS =====

@router.message(Command("progress"))
async def cmd_add_progress(message: Message, state: FSMContext):
    if not is_authenticated(message.from_user.id):
        await message.answer(msg('auth.auth_required'))
        return

    args = message.text.split()

    if len(args) >= 2:
        try:
            challenge_id = int(args[1])
            await start_progress_flow(message, state, challenge_id)
        except ValueError:
            await message.answer(msg('errors.invalid_input'))
        return

    # Нет ID — показываем список активных челленджей и ждем ввод
    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/challenges", headers=headers) as resp:
                if resp.status == 200:
                    challenges = await resp.json()
                    active = [c for c in challenges if c['status'] in ['pending', 'active']]

                    if not active:
                        await message.answer(msg('progress.no_active'))
                        return

                    text = "📋 *Активные челленджи:*\n\n"
                    for ch in active[:10]:
                        status_emoji = {'pending': '⏳', 'active': '🔥'}.get(ch['status'], '❓')
                        text += f"{status_emoji} `{ch['id']}` — {ch['title']}\n"
                    text += "\n✏️ Введи ID челленджа для добавления прогресса:"
                    
                    await message.answer(text, parse_mode="Markdown")
                    await state.set_state(WaitingChallengeIdStates.waiting_for_progress_id)
                else:
                    await message.answer(msg('challenges.fetch_error'))
        except Exception as e:
            await message.answer(msg('errors.generic'))

@router.message(StateFilter(WaitingChallengeIdStates.waiting_for_progress_id))
async def process_progress_id_input(message: Message, state: FSMContext):
    try:
        challenge_id = int(message.text.strip())
        await start_progress_flow(message, state, challenge_id)
    except ValueError:
        await state.clear()
        await message.answer(msg('errors.invalid_input'))

async def start_progress_flow(message: Message, state: FSMContext, challenge_id: int):
    await state.update_data(challenge_id=challenge_id)

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        async with session.get(f"{API_URL}/challenges/{challenge_id}", headers=headers) as resp:
            if resp.status != 200:
                await message.answer(msg('challenges.not_found'))
                await state.clear()
                return

            ch = await resp.json()
            await state.update_data(challenge_type=ch['type'])

            if ch['type'] == 'accumulative':
                await message.answer(msg('progress.ask_value_accumulative', challenge_title=ch['title']))
            else:
                await message.answer(msg('progress.ask_value_consistency', challenge_title=ch['title']))

            await state.set_state(AddProgressStates.waiting_for_value)

@router.message(StateFilter(AddProgressStates.waiting_for_value))
async def process_progress_value(message: Message, state: FSMContext):
    from notifications import notify_challenge_progress

    data = await state.get_data()
    challenge_type = data.get('challenge_type')

    if challenge_type == 'accumulative':
        try:
            value = int(message.text.strip())
            if value <= 0:
                raise ValueError
        except ValueError:
            await message.answer(msg('progress.invalid_value'))
            return
    else:
        value = 1

    token = get_token(message.from_user.id)
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(f"{API_URL}/profile", headers=headers) as profile_resp:
                if profile_resp.status != 200:
                    await message.answer(msg('errors.generic'))
                    return
                user_profile = await profile_resp.json()
                user_id = user_profile['id']

            async with session.post(
                f"{API_URL}/challenges/{data['challenge_id']}/progress",
                headers=headers,
                json={"value": value}
            ) as resp:
                if resp.status == 200:
                    await message.answer(msg('progress.added', value=value))

                    await notify_challenge_progress(
                        bot=message.bot,
                        challenge_id=data['challenge_id'],
                        actor_user_id=user_id,
                        actor_telegram_id=message.from_user.id,
                        value=value,
                        challenge_type=challenge_type,
                        token=token
                    )
                else:
                    error = await resp.text()
                    await message.answer(msg('progress.error', error=error))
        except Exception as e:
            await message.answer(msg('errors.generic'))

    await state.clear()

