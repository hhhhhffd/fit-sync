/**
 * Message loader with auto-context placeholders
 * 
 * Usage:
 *   // After login, set user context once:
 *   setUserContext(userData);
 *   
 *   // Then all placeholders auto-filled:
 *   msg('auth.welcome_back')  // {name}, {wins}, etc auto-filled
 *   
 *   // Or override specific values:
 *   msg('challenges.created', {challenge_title: 'My Challenge'})
 */

let messages = {};
let context = {};

async function loadMessages() {
    try {
        const response = await fetch('/messages.yaml', {
            headers: { 'ngrok-skip-browser-warning': 'true' }
        });
        const yaml = await response.text();
        messages = parseYaml(yaml);
    } catch (e) {
        console.error('Failed to load messages:', e);
    }
}

// Simple YAML parser (for flat/nested strings only)
function parseYaml(yaml) {
    const result = {};
    const lines = yaml.split('\n');
    const stack = [{ obj: result, indent: -2 }];
    
    for (let line of lines) {
        if (line.trim().startsWith('#') || !line.trim()) continue;
        
        const indent = line.search(/\S/);
        const content = line.trim();
        
        if (content.endsWith('|')) {
            const key = content.slice(0, -1).replace(':', '').trim();
            while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
                stack.pop();
            }
            const parent = stack[stack.length - 1].obj;
            parent[key] = '';
            stack.push({ obj: parent, indent, key, multiline: true });
            continue;
        }
        
        if (stack[stack.length - 1].multiline && indent > stack[stack.length - 1].indent) {
            const parent = stack[stack.length - 1].obj;
            const key = stack[stack.length - 1].key;
            parent[key] += (parent[key] ? '\n' : '') + content;
            continue;
        }
        
        if (stack[stack.length - 1].multiline) {
            stack.pop();
        }
        
        while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
            stack.pop();
        }
        
        if (content.includes(':')) {
            const colonIndex = content.indexOf(':');
            const key = content.slice(0, colonIndex).trim();
            const value = content.slice(colonIndex + 1).trim();
            
            const parent = stack[stack.length - 1].obj;
            
            if (value === '' || value === '|') {
                parent[key] = {};
                stack.push({ obj: parent[key], indent });
            } else {
                parent[key] = value.replace(/^["']|["']$/g, '');
            }
        }
    }
    
    return result;
}

/**
 * Set user context for auto-fill (call after login)
 */
function setUserContext(user) {
    if (!user) return;
    Object.assign(context, {
        user_id: user.id || '',
        name: user.name || user.first_name || 'Атлет',
        username: user.username || user.login || '',
        login: user.login || user.username || '',
        wins: user.total_wins || 0,
        streak: user.current_streak || 0,
        best_streak: user.best_streak || 0,
    });
}

/**
 * Set challenge context for auto-fill
 */
function setChallengeContext(challenge) {
    if (!challenge) return;
    Object.assign(context, {
        challenge_id: challenge.id || '',
        challenge_title: challenge.title || '',
        goal: challenge.goal_value || 0,
        type: challenge.type || 'accumulative',
        invite_code: challenge.invite_code || '',
        start_date: challenge.start_date || '',
        end_date: challenge.end_date || '',
        participants_count: (challenge.participants || []).length,
    });
}

/**
 * Set custom context values
 */
function setContext(values) {
    Object.assign(context, values);
}

/**
 * Clear all context
 */
function clearContext() {
    context = {};
}

/**
 * Get auto-generated context (date, time)
 */
function getAutoContext() {
    const now = new Date();
    return {
        date: now.toLocaleDateString('ru-RU'),
        time: now.toLocaleTimeString('ru-RU', {hour: '2-digit', minute: '2-digit'}),
        year: now.getFullYear(),
        month: now.getMonth() + 1,
        day: now.getDate(),
    };
}

/**
 * Get message by dot-notation key with auto-filled placeholders
 */
function msg(key, params = {}) {
    const parts = key.split('.');
    let value = messages;
    
    for (const part of parts) {
        if (value && typeof value === 'object' && part in value) {
            value = value[part];
        } else {
            return `[${key}]`;
        }
    }
    
    if (typeof value !== 'string') {
        return `[${key}]`;
    }
    
    // Merge all contexts (params > context > auto)
    const allContext = {
        ...getAutoContext(),
        ...context,
        ...params
    };
    
    // Replace placeholders {name} -> allContext.name
    return value.replace(/\{(\w+)\}/g, (match, name) => {
        return allContext[name] !== undefined ? allContext[name] : match;
    });
}

// Auto-load on page load
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', loadMessages);
} else {
    loadMessages();
}

// Auto-set user context from localStorage if available
try {
    const savedUser = localStorage.getItem('user');
    if (savedUser) {
        setUserContext(JSON.parse(savedUser));
    }
} catch (e) {}
