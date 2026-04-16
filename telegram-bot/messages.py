"""
Message loader with placeholder support
Usage:
    from messages import msg, set_user_context
    
    # Set current user context (call once after auth)
    set_user_context(user_dict)
    
    # Then all placeholders auto-filled
    text = msg('auth.welcome_new')  # {name}, {wins}, etc auto-filled
"""

import yaml
from pathlib import Path
from datetime import datetime


class Messages:
    def __init__(self):
        self._messages = {}
        self._context = {}
        self._load()
    
    def _load(self):
        path = Path(__file__).parent / 'messages.yaml'
        if path.exists():
            with open(path, 'r', encoding='utf-8') as f:
                self._messages = yaml.safe_load(f) or {}
    
    def reload(self):
        """Reload messages from file (hot reload)"""
        self._load()
    
    def set_context(self, **kwargs):
        """Set global context that auto-fills placeholders"""
        self._context.update(kwargs)
    
    def set_user(self, user: dict):
        """Set user context from user dict (from API)"""
        if not user:
            return
        self._context.update({
            'user_id': user.get('id', ''),
            'name': user.get('name') or user.get('first_name') or 'Атлет',
            'username': user.get('username') or user.get('login') or '',
            'login': user.get('login') or user.get('username') or '',
            'first_name': user.get('first_name') or user.get('name') or '',
            'wins': user.get('total_wins', 0),
            'streak': user.get('current_streak', 0),
            'best_streak': user.get('best_streak', 0),
            'telegram_id': user.get('telegram_id', ''),
        })
    
    def set_challenge(self, challenge: dict):
        """Set challenge context"""
        if not challenge:
            return
        self._context.update({
            'challenge_id': challenge.get('id', ''),
            'challenge_title': challenge.get('title', ''),
            'goal': challenge.get('goal_value', 0),
            'type': challenge.get('type', 'accumulative'),
            'invite_code': challenge.get('invite_code', ''),
            'start_date': challenge.get('start_date', ''),
            'end_date': challenge.get('end_date', ''),
            'participants_count': len(challenge.get('participants', [])),
        })
    
    def clear_context(self):
        """Clear all context"""
        self._context = {}
    
    def _get_auto_context(self) -> dict:
        """Get auto-generated context (date, time, etc)"""
        now = datetime.now()
        return {
            'date': now.strftime('%d.%m.%Y'),
            'time': now.strftime('%H:%M'),
            'year': now.year,
            'month': now.month,
            'day': now.day,
        }
    
    def get(self, key: str, **kwargs) -> str:
        """
        Get message by dot-notation key and replace placeholders
        
        Placeholders are filled from:
        1. kwargs (highest priority)
        2. context (set_user, set_challenge)
        3. auto context (date, time)
        """
        # Navigate to nested key
        parts = key.split('.')
        value = self._messages
        
        for part in parts:
            if isinstance(value, dict) and part in value:
                value = value[part]
            else:
                return f"[{key}]"
        
        if not isinstance(value, str):
            return f"[{key}]"
        
        # Merge all contexts (kwargs override context override auto)
        all_context = {
            **self._get_auto_context(),
            **self._context,
            **kwargs
        }
        
        # Replace placeholders
        try:
            return value.format(**all_context)
        except KeyError:
            # Some placeholders missing - replace what we can
            import re
            def replacer(match):
                key = match.group(1)
                return str(all_context.get(key, match.group(0)))
            return re.sub(r'\{(\w+)\}', replacer, value)
    
    def __call__(self, key: str, **kwargs) -> str:
        return self.get(key, **kwargs)


# Singleton instance
_instance = None

def _get_instance() -> Messages:
    global _instance
    if _instance is None:
        _instance = Messages()
    return _instance

def msg(key: str, **kwargs) -> str:
    """Get message by key with auto-filled placeholders"""
    return _get_instance().get(key, **kwargs)

def set_user_context(user: dict):
    """Set current user for auto-fill (call after auth)"""
    _get_instance().set_user(user)

def set_challenge_context(challenge: dict):
    """Set current challenge for auto-fill"""
    _get_instance().set_challenge(challenge)

def set_context(**kwargs):
    """Set custom context values"""
    _get_instance().set_context(**kwargs)

def clear_context():
    """Clear all context"""
    _get_instance().clear_context()

def reload_messages():
    """Reload messages from file"""
    _get_instance().reload()

def btn(key: str) -> str:
    """Get button text"""
    return msg(f'buttons.{key}')
