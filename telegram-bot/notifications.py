"""
Модуль уведомлений для Telegram бота
Реализует:
1. Уведомления соперников при обновлении прогресса
2. Утренние напоминания о челленджах
"""

import asyncio
import logging
from datetime import datetime, time, timedelta
from typing import Optional
import aiohttp
from aiogram import Bot

from config import API_URL

logger = logging.getLogger(__name__)


async def notify_challenge_progress(
    bot: Bot,
    challenge_id: int,
    actor_user_id: int,
    actor_telegram_id: int,
    value: int,
    challenge_type: str,
    token: str
):
    """
    Уведомляет соперников когда кто-то добавил прогресс

    Args:
        bot: Экземпляр aiogram Bot
        challenge_id: ID челленджа
        actor_user_id: ID пользователя который добавил прогресс
        actor_telegram_id: Telegram ID актора (чтобы не отправлять ему)
        value: Добавленное значение
        challenge_type: Тип челленджа (accumulative/consistency)
        token: JWT токен для API запросов
    """
    headers = {"Authorization": f"Bearer {token}"}

    try:
        # Получаем данные челленджа и участников
        async with aiohttp.ClientSession() as session:
            async with session.get(
                f"{API_URL}/challenges/{challenge_id}",
                headers=headers
            ) as resp:
                if resp.status != 200:
                    logger.error(f"Failed to fetch challenge {challenge_id}: status {resp.status}")
                    return

                challenge = await resp.json()

            # Получаем данные актора (профиль пользователя который добавил прогресс)
            async with session.get(
                f"{API_URL}/profile",
                headers=headers
            ) as resp:
                if resp.status != 200:
                    logger.warning(f"Failed to fetch actor profile: status {resp.status}")
                    actor_name = "Соперник"
                else:
                    actor_profile = await resp.json()
                    actor_name = actor_profile.get('name', 'Соперник')

        # Получаем список участников из челленджа
        participants = challenge.get('participants', [])

        logger.info(f"Challenge {challenge_id} has {len(participants)} participants")

        # Формируем текст в зависимости от типа челленджа
        if challenge_type == 'accumulative':
            # "Алекс добавил +20. Твой ход."
            notification_text = f"🔥 {actor_name} добавил +{value}. Твой ход."
        else:
            # Для consistency
            notification_text = f"✅ {actor_name} отметился сегодня. Не отставай!"

        # Отправляем уведомления всем участникам кроме актора
        notifications_sent = 0
        for participant in participants:
            participant_tg_id = participant.get('telegram_id')
            participant_name = participant.get('name', 'Unknown')

            logger.debug(f"Checking participant: {participant_name} (user_id={participant.get('id')}, telegram_id={participant_tg_id})")

            # Пропускаем актора и тех у кого нет telegram_id
            if not participant_tg_id:
                logger.warning(f"Participant {participant_name} has no telegram_id, skipping")
                continue

            if participant_tg_id == actor_telegram_id:
                logger.debug(f"Skipping actor {participant_name}")
                continue

            try:
                # Формируем полное сообщение с кнопкой
                full_text = (
                    f"{notification_text}\n\n"
                    f"🏆 {challenge['title']}\n"
                    f"📊 Добавить прогресс: /progress {challenge_id}"
                )

                await bot.send_message(
                    chat_id=participant_tg_id,
                    text=full_text
                )
                notifications_sent += 1
                logger.info(f"✓ Sent progress notification to {participant_name} (telegram_id={participant_tg_id})")
            except Exception as e:
                logger.error(f"✗ Failed to send notification to {participant_name} (telegram_id={participant_tg_id}): {e}")

        if notifications_sent == 0:
            logger.warning(f"No notifications sent for challenge {challenge_id}. Total participants: {len(participants)}, actor_telegram_id: {actor_telegram_id}")

    except Exception as e:
        logger.error(f"Error in notify_challenge_progress: {e}")


async def send_daily_reminders(bot: Bot):
    """
    Отправляет утренние напоминания всем пользователям с активными челленджами
    Должна вызываться из планировщика (scheduler)
    """
    logger.info("Starting daily reminders task")

    # Здесь нужен токен администратора или системный токен
    # Для MVP можно хранить в config или получать через специальный endpoint
    # Пока пропускаем авторизацию для системных задач

    try:
        async with aiohttp.ClientSession() as session:
            # Получаем список всех активных челленджей
            # Примечание: нужен endpoint для получения всех челленджей (admin)
            # Для MVP можно хранить список telegram_id пользователей с активными челленджами

            # Временное решение: используем GET /challenges без авторизации
            # В production нужен системный токен
            logger.warning("Daily reminders require system token - not fully implemented")

            # TODO: Реализовать получение пользователей с активными челленджами
            # через admin endpoint или отдельную таблицу

    except Exception as e:
        logger.error(f"Error in send_daily_reminders: {e}")


async def send_morning_reminder_to_user(bot: Bot, telegram_id: int, challenges: list):
    """
    Отправляет утреннее напоминание конкретному пользователю

    Args:
        bot: Экземпляр aiogram Bot
        telegram_id: Telegram ID пользователя
        challenges: Список активных челленджей пользователя
    """
    if not challenges:
        return

    try:
        # Формируем текст напоминания
        text = "☀️ Доброе утро! Не забудь про свои цели сегодня:\n\n"

        for ch in challenges[:3]:  # Максимум 3 челленджа в напоминании
            type_emoji = '📊' if ch.get('type') == 'accumulative' else '📅'
            text += f"{type_emoji} {ch['title']}\n"

        text += f"\n💪 Добавить прогресс: /progress"

        await bot.send_message(
            chat_id=telegram_id,
            text=text
        )
        logger.info(f"Sent morning reminder to user {telegram_id}")
    except Exception as e:
        logger.error(f"Failed to send morning reminder to {telegram_id}: {e}")


class DailyReminderScheduler:
    """
    Планировщик для утренних напоминаний
    Запускается как фоновая задача asyncio
    """

    def __init__(self, bot: Bot, reminder_time: time = time(8, 0)):
        """
        Args:
            bot: Экземпляр aiogram Bot
            reminder_time: Время отправки напоминаний (по умолчанию 08:00)
        """
        self.bot = bot
        self.reminder_time = reminder_time
        self._task: Optional[asyncio.Task] = None

    async def _run_scheduler(self):
        """Основной цикл планировщика"""
        logger.info(f"Daily reminder scheduler started (time: {self.reminder_time})")

        while True:
            try:
                now = datetime.now()
                target = datetime.combine(now.date(), self.reminder_time)

                # Если время уже прошло сегодня, планируем на завтра
                if now >= target:
                    target = datetime.combine(
                        now.date() + timedelta(days=1),
                        self.reminder_time
                    )

                # Вычисляем время до следующего запуска
                wait_seconds = (target - now).total_seconds()
                logger.info(f"Next reminder in {wait_seconds / 3600:.1f} hours")

                # Ждём до нужного времени
                await asyncio.sleep(wait_seconds)

                # Отправляем напоминания
                await send_daily_reminders(self.bot)

                # Ждём минуту чтобы не запускаться повторно в ту же минуту
                await asyncio.sleep(60)

            except asyncio.CancelledError:
                logger.info("Daily reminder scheduler stopped")
                break
            except Exception as e:
                logger.error(f"Error in scheduler loop: {e}")
                # Ждём минуту перед повторной попыткой
                await asyncio.sleep(60)

    def start(self):
        """Запускает планировщик"""
        if self._task is None or self._task.done():
            self._task = asyncio.create_task(self._run_scheduler())
            logger.info("Daily reminder scheduler task created")

    def stop(self):
        """Останавливает планировщик"""
        if self._task and not self._task.done():
            self._task.cancel()
            logger.info("Daily reminder scheduler stopping")


# Простой in-memory кэш для хранения telegram_id пользователей
# В production лучше использовать Redis или хранить в БД
_user_sessions = {}


def cache_user_session(user_id: int, telegram_id: int, token: str):
    """Кэширует данные сессии пользователя для уведомлений"""
    _user_sessions[user_id] = {
        'telegram_id': telegram_id,
        'token': token,
        'cached_at': datetime.now()
    }


def get_cached_user_session(user_id: int) -> Optional[dict]:
    """Получает закэшированные данные пользователя"""
    return _user_sessions.get(user_id)

