// Auto-detect API URL based on current location
const API_URL = window.location.hostname === 'localhost'
    ? 'http://localhost:8080/api'
    : window.location.origin + '/api';

// Check if opened in Telegram
function checkTelegramOnly() {
    // Allow localhost for development
    if (window.location.hostname === 'localhost') {
        return true;
    }

    // Check if Telegram Web App
    const isTelegram = window.Telegram && window.Telegram.WebApp && window.Telegram.WebApp.initData;

    if (!isTelegram) {
        // Not from Telegram - block access
        document.body.innerHTML = `
            <div style="display: flex; align-items: center; justify-content: center; min-height: 100vh; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;">
                <div style="text-align: center; color: white; max-width: 500px; padding: 40px;">
                    <div style="font-size: 80px; margin-bottom: 20px;">🔒</div>
                    <h1 style="font-size: 32px; margin: 0 0 20px 0;">Доступ только через Telegram</h1>
                    <p style="font-size: 18px; margin: 0 0 30px 0; opacity: 0.9;">
                        Этот сайт работает только как Telegram Web App.<br>
                        Открой его через бота!
                    </p>
                    <a href="https://t.me/Ghostty_best_terminal_bot?start=webapp"
                       style="display: inline-block; background: white; color: #667eea; padding: 15px 40px; border-radius: 12px; text-decoration: none; font-weight: bold; font-size: 18px; box-shadow: 0 4px 20px rgba(0,0,0,0.2);">
                        📱 Открыть в Telegram
                    </a>
                </div>
            </div>
        `;
        throw new Error('Access denied: Not opened from Telegram');
    }

    return true;
}

// Auth utilities
function getToken() {
    return localStorage.getItem('token');
}

function getUser() {
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
}

function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.location.href = 'login.html';
}

function checkAuth() {
    if (!getToken()) {
        window.location.href = 'login.html';
    }
}

// API call wrapper
async function apiCall(endpoint, options = {}) {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        'ngrok-skip-browser-warning': 'true',
        ...(token && { 'Authorization': `Bearer ${token}` })
    };

    const response = await fetch(`${API_URL}${endpoint}`, {
        ...options,
        headers: { ...headers, ...options.headers }
    });

    if (response.status === 401) {
        logout();
        return;
    }

    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }

    return response.json();
}

// Utility functions
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('ru-RU', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}

function showMessage(elementId, text, type) {
    const element = document.getElementById(elementId);
    if (element) {
        element.className = type;
        element.textContent = text;
        element.classList.remove('hidden');
        setTimeout(() => element.classList.add('hidden'), 3000);
    }
}
