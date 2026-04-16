// Telegram Web App initialization
let tg;

// Wait for messages to load
async function initWebApp() {
    // Ждём загрузки сообщений
    await loadMessages();
    
    // Обновляем тексты навигации из конфига
    updateNavTexts();
    
    if (!window.Telegram || !window.Telegram.WebApp) {
        document.getElementById('loading').innerHTML = `
            <h2>${msg('common.error')}</h2>
            <p>${msg('auth.errors.network')}</p>
        `;
        return;
    }
    
    tg = window.Telegram.WebApp;
    tg.expand();
    tg.ready();

    const user = tg.initDataUnsafe.user;

    if (!user) {
        document.getElementById('loading').innerHTML = `
            <h2>${msg('common.error')}</h2>
            <p>${msg('auth.errors.network')}</p>
        `;
        return;
    }

    authenticateUser(user);
}

function updateNavTexts() {
    const nav = document.getElementById('nav-dashboard');
    if (nav) nav.textContent = msg('navigation.dashboard');
    
    const navProfile = document.getElementById('nav-profile');
    if (navProfile) navProfile.textContent = msg('navigation.profile');
    
    const navChallenges = document.getElementById('nav-challenges');
    if (navChallenges) navChallenges.textContent = msg('navigation.challenges');
    
    const navLeaderboard = document.getElementById('nav-leaderboard');
    if (navLeaderboard) navLeaderboard.textContent = msg('navigation.leaderboard');
    
    // Loading texts
    const loadingTitle = document.getElementById('loading-title');
    if (loadingTitle) loadingTitle.textContent = msg('common.loading');
}

async function authenticateUser(user) {
    try {
        const response = await fetch(`${API_URL}/telegram-auth`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'ngrok-skip-browser-warning': 'true'
            },
            body: JSON.stringify({
                telegram_id: user.id,
                first_name: user.first_name,
                last_name: user.last_name,
                username: user.username,
                init_data: tg.initData
            })
        });

        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('token', data.token);
            localStorage.setItem('user', JSON.stringify(data.user));
            setUserContext(data.user);

            document.getElementById('loading').classList.add('hidden');
            document.getElementById('app').classList.remove('hidden');

            showDashboard();
        } else {
            showPhoneAuth();
        }
    } catch (error) {
        console.error('Auth error:', error);
        showPhoneAuth();
    }
}

function showPhoneAuth() {
    document.getElementById('loading').innerHTML = `
        <h2>${msg('common.error')}</h2>
        <p>${msg('auth.errors.network')}</p>
        <button onclick="tg.close()">${msg('common.back')}</button>
    `;
}

async function showDashboard() {
    try {
        const user = await apiCall('/profile');
        setUserContext(user);

        document.getElementById('content').innerHTML = `
            <h2>${msg('dashboard.title')}</h2>

            <div class="stats">
                <div class="stat-card">
                    <h3>${msg('dashboard.stats.wins')}</h3>
                    <div class="value">${user.total_wins}</div>
                </div>
                <div class="stat-card">
                    <h3>${msg('dashboard.stats.streak')}</h3>
                    <div class="value">${user.current_streak}</div>
                </div>
                <div class="stat-card">
                    <h3>${msg('dashboard.stats.best_streak')}</h3>
                    <div class="value">${user.best_streak}</div>
                </div>
            </div>

            <div class="card">
                <h3>${msg('dashboard.activities.title')}</h3>
                <button onclick="addActivityModal()" class="mt-1">➕ ${msg('activity_types.other')}</button>
                <button onclick="showChallenges()" class="mt-1">🏆 ${msg('navigation.challenges')}</button>
            </div>
        `;
    } catch (error) {
        showError(msg('auth.errors.network'));
    }
}

async function showProfile() {
    try {
        const user = await apiCall('/profile');
        setUserContext(user);

        document.getElementById('content').innerHTML = `
            <h2>${msg('profile.title')}</h2>

            <div class="card">
                <div class="form-group">
                    <label>${msg('profile.form.name')}</label>
                    <input type="text" id="name" value="${user.name || ''}" placeholder="${msg('profile.form.name_placeholder')}">
                </div>
                <div class="form-group">
                    <label>${msg('profile.form.height')}</label>
                    <input type="number" id="height" value="${user.height || ''}">
                </div>
                <div class="form-group">
                    <label>${msg('profile.form.weight')}</label>
                    <input type="number" id="weight" value="${user.weight || ''}">
                </div>
                <button onclick="updateProfile()">${msg('profile.save_button')}</button>
            </div>

            <div class="card">
                <h3>${msg('profile.achievements_title')}</h3>
                <div id="achievements"></div>
            </div>
        `;

        loadAchievements();
    } catch (error) {
        showError(msg('auth.errors.network'));
    }
}

async function updateProfile() {
    const profile = {
        name: document.getElementById('name').value,
        height: parseFloat(document.getElementById('height').value) || 0,
        weight: parseFloat(document.getElementById('weight').value) || 0
    };

    try {
        await apiCall('/profile', {
            method: 'PUT',
            body: JSON.stringify(profile)
        });

        tg.showAlert(msg('profile.updated'));
    } catch (error) {
        tg.showAlert(msg('auth.errors.network'));
    }
}

async function showChallenges() {
    try {
        const challenges = await apiCall('/challenges');

        let html = `
            <div class="challenges-header">
                <h2>${msg('challenges.title')}</h2>
                <div class="challenges-actions">
                    <button onclick="showCreateChallenge()" class="btn-primary">➕ Создать</button>
                    <button onclick="showJoinChallenge()" class="btn-secondary">🔗 Присоединиться</button>
                </div>
            </div>
        `;

        if (!challenges || challenges.length === 0) {
            html += `<p class="empty-state">${msg('challenges.empty')}</p>`;
        } else {
            html += '<div class="challenges-list">';
            challenges.forEach(ch => {
                setChallengeContext(ch);
                const statusEmoji = {'pending': '⏳', 'active': '🔥', 'completed': '✅'}[ch.status] || '❓';
                const typeEmoji = ch.type === 'accumulative' ? '📊' : '📅';
                
                html += `
                    <div class="card challenge-card" onclick="showChallengeDetail(${ch.id})">
                        <div class="challenge-header">
                            <span class="challenge-status ${ch.status}">${statusEmoji}</span>
                            <span class="challenge-type">${typeEmoji}</span>
                        </div>
                        <h3>${ch.title}</h3>
                        <p>${ch.description || 'Без описания'}</p>
                        <div class="challenge-meta">
                            <span>🎯 ${ch.goal_value || 0}</span>
                            <span>👥 ${(ch.participants || []).length}</span>
                        </div>
                    </div>
                `;
            });
            html += '</div>';
        }

        document.getElementById('content').innerHTML = html;
    } catch (error) {
        console.error('Challenges error:', error);
        document.getElementById('content').innerHTML = `
            <div class="card">
                <h3>${msg('common.error')}</h3>
                <p>${msg('auth.errors.network')}</p>
                <button onclick="showChallenges()">${msg('common.back')}</button>
            </div>
        `;
    }
}

// Показать детали челленджа
async function showChallengeDetail(challengeId) {
    try {
        const ch = await apiCall(`/challenges/${challengeId}`);
        let progressData = [];
        try {
            progressData = await apiCall(`/challenges/${challengeId}/progress`);
        } catch(e) {}
        
        setChallengeContext(ch);
        
        const statusEmoji = {'pending': '⏳', 'active': '🔥', 'completed': '✅'}[ch.status] || '❓';
        const typeText = ch.type === 'accumulative' ? 'Накопительный' : 'Последовательный';
        const startDate = new Date(ch.start_date).toLocaleDateString('ru-RU');
        const endDate = new Date(ch.end_date).toLocaleDateString('ru-RU');
        
        let html = `
            <button onclick="showChallenges()" class="btn-back">← Назад</button>
            
            <div class="challenge-detail">
                <div class="challenge-detail-header">
                    <h2>${statusEmoji} ${ch.title}</h2>
                </div>
                
                <div class="card">
                    <p>${ch.description || 'Без описания'}</p>
                    <div class="challenge-info-grid">
                        <div><strong>Тип:</strong> ${typeText}</div>
                        <div><strong>Цель:</strong> ${ch.goal_value}</div>
                        <div><strong>Период:</strong> ${startDate} - ${endDate}</div>
                        <div><strong>Участников:</strong> ${(ch.participants || []).length}</div>
                    </div>
                    ${ch.invite_code ? `<div class="invite-code"><strong>Код:</strong> <code>${ch.invite_code}</code></div>` : ''}
                </div>
        `;
        
        // Прогресс участников
        if (progressData && progressData.length > 0) {
            html += `<div class="card"><h3>👥 Прогресс участников</h3><div class="progress-list">`;
            progressData.forEach(p => {
                const participant = (ch.participants || []).find(u => u.id === p.user_id);
                const name = participant?.name || `User #${p.user_id}`;
                const percent = ch.goal_value > 0 ? Math.min(100, Math.round(p.current_progress / ch.goal_value * 100)) : 0;
                
                html += `
                    <div class="progress-item">
                        <div class="progress-name">${name}</div>
                        <div class="progress-bar-container">
                            <div class="progress-bar" style="width: ${percent}%"></div>
                        </div>
                        <div class="progress-value">${p.current_progress}/${ch.goal_value}</div>
                    </div>
                `;
            });
            html += `</div></div>`;
        }
        
        // Действия
        if (ch.status !== 'completed') {
            html += `
                <div class="card challenge-actions">
                    <h3>⚡ Действия</h3>
                    <button onclick="showAddProgress(${ch.id}, '${ch.type}')" class="btn-primary">➕ Добавить прогресс</button>
                </div>
            `;
        }
        
        html += `</div>`;
        
        document.getElementById('content').innerHTML = html;
    } catch (error) {
        console.error('Challenge detail error:', error);
        showError('Не удалось загрузить челлендж');
    }
}

// Форма создания челленджа
function showCreateChallenge() {
    document.getElementById('content').innerHTML = `
        <button onclick="showChallenges()" class="btn-back">← Назад</button>
        
        <h2>➕ Создать челлендж</h2>
        
        <div class="card">
            <div class="form-group">
                <label>Название</label>
                <input type="text" id="ch-title" placeholder="Например: 100 км за неделю">
            </div>
            
            <div class="form-group">
                <label>Описание</label>
                <textarea id="ch-description" placeholder="Опиши правила челленджа..."></textarea>
            </div>
            
            <div class="form-group">
                <label>Тип</label>
                <select id="ch-type">
                    <option value="accumulative">📊 Накопительный (сумма значений)</option>
                    <option value="consistency">📅 Последовательный (дни подряд)</option>
                </select>
            </div>
            
            <div class="form-group">
                <label>Цель</label>
                <input type="number" id="ch-goal" placeholder="Например: 100">
            </div>
            
            <div class="form-group">
                <label>Длительность (дней)</label>
                <input type="number" id="ch-duration" value="7">
            </div>
            
            <button onclick="createChallenge()" class="btn-primary">🚀 Создать</button>
        </div>
    `;
}

async function createChallenge() {
    const title = document.getElementById('ch-title').value.trim();
    const description = document.getElementById('ch-description').value.trim();
    const type = document.getElementById('ch-type').value;
    const goalValue = parseInt(document.getElementById('ch-goal').value) || 0;
    const duration = parseInt(document.getElementById('ch-duration').value) || 7;
    
    if (!title) {
        tg.showAlert('Введи название челленджа');
        return;
    }
    if (goalValue <= 0) {
        tg.showAlert('Введи цель больше 0');
        return;
    }
    
    const startDate = new Date();
    const endDate = new Date();
    endDate.setDate(endDate.getDate() + duration);
    
    try {
        const result = await apiCall('/challenges', {
            method: 'POST',
            body: JSON.stringify({
                title,
                description,
                type,
                goal_value: goalValue,
                start_date: startDate.toISOString(),
                end_date: endDate.toISOString()
            })
        });
        
        tg.showAlert(`Челлендж создан!\nКод: ${result.invite_code}`);
        showChallengeDetail(result.id);
    } catch (error) {
        tg.showAlert('Ошибка создания челленджа');
    }
}

// Присоединение к челленджу
function showJoinChallenge() {
    document.getElementById('content').innerHTML = `
        <button onclick="showChallenges()" class="btn-back">← Назад</button>
        
        <h2>🔗 Присоединиться</h2>
        
        <div class="card">
            <div class="form-group">
                <label>Код приглашения</label>
                <input type="text" id="join-code" placeholder="Введи код...">
            </div>
            
            <button onclick="joinChallenge()" class="btn-primary">🚀 Присоединиться</button>
        </div>
    `;
}

async function joinChallenge() {
    const code = document.getElementById('join-code').value.trim();
    
    if (!code) {
        tg.showAlert('Введи код приглашения');
        return;
    }
    
    try {
        await apiCall(`/challenges/join/${code}`, { method: 'POST' });
        tg.showAlert('Ты присоединился к челленджу! 💪');
        showChallenges();
    } catch (error) {
        tg.showAlert('Неверный код или челлендж недоступен');
    }
}

// Добавление прогресса
function showAddProgress(challengeId, challengeType) {
    const isAccumulative = challengeType === 'accumulative';
    
    document.getElementById('content').innerHTML = `
        <button onclick="showChallengeDetail(${challengeId})" class="btn-back">← Назад</button>
        
        <h2>➕ Добавить прогресс</h2>
        
        <div class="card">
            ${isAccumulative ? `
                <div class="form-group">
                    <label>Значение</label>
                    <input type="number" id="progress-value" placeholder="Сколько сделал?">
                </div>
            ` : `
                <p>Для последовательного челленджа засчитывается 1 день.</p>
            `}
            
            <button onclick="addProgress(${challengeId}, ${isAccumulative})" class="btn-primary">✅ Добавить</button>
        </div>
    `;
}

async function addProgress(challengeId, isAccumulative) {
    const value = isAccumulative ? (parseInt(document.getElementById('progress-value').value) || 0) : 1;
    
    if (isAccumulative && value <= 0) {
        tg.showAlert('Введи значение больше 0');
        return;
    }
    
    try {
        await apiCall(`/challenges/${challengeId}/progress`, {
            method: 'POST',
            body: JSON.stringify({ value })
        });
        
        tg.showAlert('Прогресс добавлен! 💪');
        showChallengeDetail(challengeId);
    } catch (error) {
        tg.showAlert('Ошибка добавления прогресса');
    }
}

async function showLeaderboard() {
    try {
        const leaderboard = await apiCall('/leaderboard?type=wins');

        let html = `<h2>${msg('leaderboard.title')}</h2>`;

        if (!leaderboard || leaderboard.length === 0) {
            html += `<p>${msg('leaderboard.empty')}</p>`;
        } else {
            leaderboard.slice(0, 10).forEach(entry => {
                let medal = '';
                if (entry.rank === 1) medal = '🥇';
                else if (entry.rank === 2) medal = '🥈';
                else if (entry.rank === 3) medal = '🥉';

                html += `
                    <div class="leaderboard-item">
                        <div class="rank">${medal} #${entry.rank}</div>
                        <div class="user-info">
                            <h4>${entry.name || 'User #' + entry.id}</h4>
                        </div>
                        <div class="score">${entry.total_wins}</div>
                    </div>
                `;
            });
        }

        document.getElementById('content').innerHTML = html;
    } catch (error) {
        console.error('Leaderboard error:', error);
        document.getElementById('content').innerHTML = `
            <div class="card">
                <h3>${msg('common.error')}</h3>
                <p>${msg('auth.errors.network')}</p>
                <button onclick="showLeaderboard()">${msg('common.back')}</button>
            </div>
        `;
    }
}

async function loadAchievements() {
    try {
        const achievements = await apiCall('/achievements');
        const container = document.getElementById('achievements');

        if (!achievements || achievements.length === 0) {
            container.innerHTML = `<p>${msg('dashboard.achievements.empty')}</p>`;
            return;
        }

        container.innerHTML = '<div class="achievements-grid">' +
            achievements.map(a => `
                <div class="achievement">
                    <div class="icon">${a.achievement.icon}</div>
                    <h4>${a.achievement.name}</h4>
                </div>
            `).join('') +
            '</div>';
    } catch (error) {
        console.error('Failed to load achievements');
    }
}

function showError(message) {
    document.getElementById('content').innerHTML = `
        <div class="card">
            <h3>${msg('common.error')}</h3>
            <p>${message}</p>
            <button onclick="showDashboard()">${msg('common.back')}</button>
        </div>
    `;
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initWebApp);
} else {
    initWebApp();
}
