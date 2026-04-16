#!/usr/bin/env python3
"""
Тестовый скрипт для проверки системы уведомлений
Симулирует отправку уведомления без запуска полного бота
"""

import asyncio
import sys
import os

# Добавляем путь к telegram-bot модулям
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'telegram-bot'))

async def test_notifications():
    """Тестирование модуля уведомлений"""

    print("🧪 Тестирование системы уведомлений\n")

    # 1. Проверка импортов
    print("1️⃣ Проверка импортов...")
    try:
        from notifications import notify_challenge_progress, DailyReminderScheduler
        print("   ✅ Модуль notifications импортируется")
    except ImportError as e:
        print(f"   ❌ Ошибка импорта: {e}")
        return False

    # 2. Проверка конфигурации
    print("\n2️⃣ Проверка конфигурации...")
    try:
        from config import BOT_TOKEN, API_URL
        print(f"   ✅ BOT_TOKEN: {'*' * 20}{BOT_TOKEN[-10:]}")
        print(f"   ✅ API_URL: {API_URL}")
    except ImportError as e:
        print(f"   ❌ Ошибка конфига: {e}")
        return False

    # 3. Проверка что бот может быть создан
    print("\n3️⃣ Создание экземпляра бота...")
    try:
        from aiogram import Bot
        bot = Bot(token=BOT_TOKEN)
        print("   ✅ Бот создан успешно")
    except Exception as e:
        print(f"   ❌ Ошибка создания бота: {e}")
        return False

    # 4. Проверка планировщика
    print("\n4️⃣ Проверка планировщика...")
    try:
        from datetime import time
        scheduler = DailyReminderScheduler(bot, reminder_time=time(8, 0))
        print("   ✅ Планировщик создан")
        print(f"   ℹ️  Время напоминаний: 08:00")
    except Exception as e:
        print(f"   ❌ Ошибка создания планировщика: {e}")
        return False

    # 5. Проверка доступности API
    print("\n5️⃣ Проверка доступности backend API...")
    try:
        import aiohttp
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{API_URL}/../", timeout=aiohttp.ClientTimeout(total=3)) as resp:
                if resp.status in [200, 404]:  # 404 - это нормально для корня
                    print(f"   ✅ Backend доступен (status: {resp.status})")
                else:
                    print(f"   ⚠️  Backend вернул status: {resp.status}")
    except asyncio.TimeoutError:
        print("   ❌ Backend не отвечает (timeout)")
        print("   💡 Запусти backend: cd backend && go run cmd/server/main.go")
        return False
    except Exception as e:
        print(f"   ❌ Ошибка подключения к backend: {e}")
        return False

    print("\n" + "="*60)
    print("✅ Все проверки пройдены!")
    print("="*60)
    print("\n💡 Чтобы протестировать уведомления:")
    print("   1. Запусти backend: cd backend && go run cmd/server/main.go")
    print("   2. Запусти бота: cd telegram-bot && python3 bot.py")
    print("   3. Создай челлендж: /createchallenge")
    print("   4. Пригласи друга: /join <код>")
    print("   5. Добавь прогресс: /progress <id>")
    print("   6. Друг получит: '🔥 Имя добавил +X. Твой ход.'\n")

    await bot.session.close()
    return True

if __name__ == "__main__":
    result = asyncio.run(test_notifications())
    sys.exit(0 if result else 1)

