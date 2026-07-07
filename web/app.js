/**
 * KankerMail Web Client Application Logic
 */

// Application State
const state = {
    activeFolder: 'inbox',
    searchQuery: '',
    emails: [],
    selectedEmail: null,
    config: {
        theme_name: 'Neon Dark',
        use_mock: true,
        html_rich: true,
        imap: {},
        smtp: {}
    },
    loading: false
};

// DOM Elements
const elements = {
    body: document.body,
    modeIndicator: document.getElementById('modeIndicator'),
    composeBtn: document.getElementById('composeBtn'),
    folderList: document.getElementById('folderList'),
    inboxUnread: document.getElementById('inboxUnread'),
    connectionStatus: document.getElementById('connectionStatus'),
    searchInput: document.getElementById('searchInput'),
    clearSearch: document.getElementById('clearSearch'),
    folderTitle: document.getElementById('folderTitle'),
    emailCount: document.getElementById('emailCount'),
    emailList: document.getElementById('emailList'),
    listLoader: document.getElementById('listLoader'),
    emptyState: document.getElementById('emptyState'),
    viewerEmptyState: document.getElementById('viewerEmptyState'),
    viewerContent: document.getElementById('viewerContent'),
    viewSubject: document.getElementById('viewSubject'),
    viewStarBtn: document.getElementById('viewStarBtn'),
    viewReplyBtn: document.getElementById('viewReplyBtn'),
    viewDeleteBtn: document.getElementById('viewDeleteBtn'),
    senderAvatar: document.getElementById('senderAvatar'),
    viewFromName: document.getElementById('viewFromName'),
    viewFromEmail: document.getElementById('viewFromEmail'),
    viewTo: document.getElementById('viewTo'),
    viewDate: document.getElementById('viewDate'),
    viewBodyMarkup: document.getElementById('viewBodyMarkup'),
    viewBodyHtml: document.getElementById('viewBodyHtml'),
    settingsContent: document.getElementById('settingsContent'),
    toastContainer: document.getElementById('toastContainer'),
    
    // Settings Elements
    mockModeToggle: document.getElementById('mockModeToggle'),
    htmlRichToggle: document.getElementById('htmlRichToggle'),
    saveSettingsBtn: document.getElementById('saveSettingsBtn'),
    imapHost: document.getElementById('imapHost'),
    imapPort: document.getElementById('imapPort'),
    imapSSL: document.getElementById('imapSSL'),
    imapUser: document.getElementById('imapUser'),
    imapPass: document.getElementById('imapPass'),
    smtpHost: document.getElementById('smtpHost'),
    smtpPort: document.getElementById('smtpPort'),
    smtpSSL: document.getElementById('smtpSSL'),
    smtpUser: document.getElementById('smtpUser'),
    smtpPass: document.getElementById('smtpPass'),

    // Compose Elements
    composeModal: document.getElementById('composeModal'),
    composeForm: document.getElementById('composeForm'),
    closeComposeBtn: document.getElementById('closeComposeBtn'),
    discardComposeBtn: document.getElementById('discardComposeBtn'),
    composeTo: document.getElementById('composeTo'),
    composeSubject: document.getElementById('composeSubject'),
    composeBody: document.getElementById('composeBody')
};

// Initialize Application
document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    await fetchConfig();
    await refreshApp();
});

// Event Listeners
function setupEventListeners() {
    // Folder list navigation
    elements.folderList.querySelectorAll('li').forEach(item => {
        item.addEventListener('click', () => {
            const folder = item.getAttribute('data-folder');
            switchFolder(folder);
        });
    });

    // Compose Modal toggle
    elements.composeBtn.addEventListener('click', () => openCompose());
    elements.closeComposeBtn.addEventListener('click', closeCompose);
    elements.discardComposeBtn.addEventListener('click', closeCompose);
    elements.composeForm.addEventListener('submit', handleSendEmail);

    // Search bar input
    let searchTimeout;
    elements.searchInput.addEventListener('input', (e) => {
        state.searchQuery = e.target.value;
        if (state.searchQuery.length > 0) {
            elements.clearSearch.style.display = 'block';
        } else {
            elements.clearSearch.style.display = 'none';
        }

        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            fetchEmails();
        }, 300);
    });

    elements.clearSearch.addEventListener('click', () => {
        elements.searchInput.value = '';
        state.searchQuery = '';
        elements.clearSearch.style.display = 'none';
        fetchEmails();
    });

    // Detail Action Buttons
    elements.viewStarBtn.addEventListener('click', handleToggleStar);
    elements.viewDeleteBtn.addEventListener('click', handleDeleteEmail);
    elements.viewReplyBtn.addEventListener('click', handleReplyEmail);

    // Settings adjustments
    elements.mockModeToggle.addEventListener('change', async (e) => {
        state.config.use_mock = e.target.checked;
        await saveSettingsAPI({ use_mock: state.config.use_mock });
        showToast(`Simulation mode ${state.config.use_mock ? 'enabled' : 'disabled'}.`);
        updateModeBanner();
        refreshApp();
    });

    elements.htmlRichToggle.addEventListener('change', async (e) => {
        state.config.html_rich = e.target.checked;
        await saveSettingsAPI({ html_rich: state.config.html_rich });
        showToast(`HTML email rendering set to ${state.config.html_rich ? 'enabled' : 'disabled'}.`);
        if (state.selectedEmail) {
            renderEmailBody(state.selectedEmail);
        }
    });

    // Theme selector cards
    document.querySelectorAll('.theme-card').forEach(card => {
        card.addEventListener('click', async () => {
            const theme = card.getAttribute('data-theme');
            const themeNameMap = {
                'neon-dark': 'Neon Dark',
                'dracula': 'Dracula',
                'catppuccin': 'Catppuccin Macchiato',
                'nord': 'Nord'
            };
            const themeName = themeNameMap[theme];
            state.config.theme_name = themeName;
            
            // Visual Update
            document.querySelectorAll('.theme-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            applyTheme(themeName);
            
            await saveSettingsAPI({ theme_name: themeName });
            showToast(`Applied Theme: ${themeName}`);
        });
    });

    // Connection configuration save
    elements.saveSettingsBtn.addEventListener('click', async () => {
        const payload = {
            imap: {
                host: elements.imapHost.value,
                port: parseInt(elements.imapPort.value) || 993,
                username: elements.imapUser.value,
                password: elements.imapPass.value,
                ssl: elements.imapSSL.checked
            },
            smtp: {
                host: elements.smtpHost.value,
                port: parseInt(elements.smtpPort.value) || 465,
                username: elements.smtpUser.value,
                password: elements.smtpPass.value,
                ssl: elements.smtpSSL.checked
            }
        };

        const result = await saveSettingsAPI(payload);
        if (result.success) {
            showToast("Server configurations saved successfully!");
        } else {
            showToast(`Failed to save credentials: ${result.error}`, true);
        }
    });
}

// Fetch Configurations
async function fetchConfig() {
    try {
        const response = await apiRequest('/api/config');
        if (response.ok) {
            state.config = await response.json();
            
            // Populate settings inputs
            elements.mockModeToggle.checked = state.config.use_mock;
            elements.htmlRichToggle.checked = state.config.html_rich;
            
            elements.imapHost.value = state.config.imap.host || '';
            elements.imapPort.value = state.config.imap.port || 993;
            elements.imapSSL.checked = state.config.imap.ssl !== false;
            elements.imapUser.value = state.config.imap.username || '';
            elements.imapPass.value = state.config.imap.password || '';

            elements.smtpHost.value = state.config.smtp.host || '';
            elements.smtpPort.value = state.config.smtp.port || 465;
            elements.smtpSSL.checked = state.config.smtp.ssl !== false;
            elements.smtpUser.value = state.config.smtp.username || '';
            elements.smtpPass.value = state.config.smtp.password || '';

            // Apply selected theme
            applyTheme(state.config.theme_name);
            updateModeBanner();
        }
    } catch (err) {
        console.error("Error loading configs:", err);
        showToast("Error loading server configs.", true);
    }
}

// Apply visual themes to <body> via CSS classes
function applyTheme(themeName) {
    elements.body.className = ''; // Reset
    
    let themeClass = 'theme-neon-dark';
    let cardSelector = 'neon-dark';
    
    switch(themeName) {
        case 'Dracula':
            themeClass = 'theme-dracula';
            cardSelector = 'dracula';
            break;
        case 'Catppuccin Macchiato':
            themeClass = 'theme-catppuccin';
            cardSelector = 'catppuccin';
            break;
        case 'Nord':
            themeClass = 'theme-nord';
            cardSelector = 'nord';
            break;
        default:
            themeClass = 'theme-neon-dark';
            cardSelector = 'neon-dark';
    }
    
    elements.body.classList.add(themeClass);
    
    // Select the card visually in setting pane
    document.querySelectorAll('.theme-card').forEach(c => {
        if (c.getAttribute('data-theme') === cardSelector) {
            c.classList.add('active');
        } else {
            c.classList.remove('active');
        }
    });
}

function updateModeBanner() {
    if (state.config.use_mock) {
        elements.modeIndicator.className = 'mode-indicator';
        elements.modeIndicator.querySelector('.indicator-text').innerText = 'Mode: Local Simulator';
    } else {
        elements.modeIndicator.className = 'mode-indicator server-sync';
        elements.modeIndicator.querySelector('.indicator-text').innerText = 'Mode: Live Sync';
    }
}

// Global folder switcher
function switchFolder(folder) {
    state.activeFolder = folder;
    
    // Update folder list active class
    elements.folderList.querySelectorAll('li').forEach(item => {
        if (item.getAttribute('data-folder') === folder) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });

    // Display appropriate title
    const folderNames = {
        inbox: 'Inbox',
        starred: 'Starred',
        sent: 'Sent',
        drafts: 'Drafts',
        archive: 'Archive',
        trash: 'Trash',
        settings: 'Settings'
    };
    elements.folderTitle.innerText = folderNames[folder] || 'Folder';
    
    // Reset selection & views
    state.selectedEmail = null;
    elements.viewerContent.style.display = 'none';
    elements.settingsContent.style.display = 'none';
    
    if (folder === 'settings') {
        elements.viewerEmptyState.style.display = 'none';
        elements.settingsContent.style.display = 'block';
        elements.emailList.innerHTML = '';
        elements.emailCount.innerText = '';
    } else {
        elements.viewerEmptyState.style.display = 'flex';
        fetchEmails();
    }
}

// Fetch lists of emails
async function fetchEmails() {
    if (state.activeFolder === 'settings') return;
    
    state.loading = true;
    elements.listLoader.style.display = 'flex';
    elements.emptyState.style.display = 'none';
    elements.emailList.innerHTML = '';
    
    try {
        let url = `/api/emails?folder=${state.activeFolder}`;
        if (state.searchQuery) {
            url += `&q=${encodeURIComponent(state.searchQuery)}`;
        }
        
        const response = await apiRequest(url);
        if (response.ok) {
            const data = await response.json();
            state.emails = data.emails || [];
            renderEmailList();
            updateUnreadCounts();
        } else {
            showToast("Failed to fetch emails.", true);
        }
    } catch (err) {
        console.error("Error fetching emails:", err);
        showToast("Server connection error.", true);
    } finally {
        state.loading = false;
        elements.listLoader.style.display = 'none';
    }
}

// Render list items in center pane
function renderEmailList() {
    elements.emailList.innerHTML = '';
    
    if (state.emails.length === 0) {
        elements.emptyState.style.display = 'flex';
        elements.emailCount.innerText = '0 messages';
        return;
    }

    elements.emptyState.style.display = 'none';
    elements.emailCount.innerText = `${state.emails.length} message${state.emails.length === 1 ? '' : 's'}`;

    state.emails.forEach((email, index) => {
        const card = document.createElement('div');
        card.className = `email-card ${email.unread ? 'unread' : ''}`;
        if (state.selectedEmail && state.selectedEmail.id === email.id) {
            card.classList.add('active');
        }

        // Render contents
        const dateStr = formatDate(email.date);
        const excerpt = email.body ? email.body.substring(0, 100) + '...' : '(No body content)';
        
        card.innerHTML = `
            <div class="card-top-row">
                <span class="card-from">${escapeHTML(email.from_name || email.from)}</span>
                <span class="card-date">${dateStr}</span>
            </div>
            <div class="card-mid-row">
                ${email.unread ? '<span class="unread-dot"></span>' : ''}
                <span class="card-subject">${escapeHTML(email.subject || '(No Subject)')}</span>
            </div>
            <div class="card-excerpt">${escapeHTML(excerpt)}</div>
            <div class="card-badges">
                <span class="star-icon ${email.starred ? 'starred' : ''}" data-id="${email.id}">★</span>
            </div>
        `;

        // Card Selection Event
        card.addEventListener('click', (e) => {
            // If click was on the star icon, ignore normal selection toggle
            if (e.target.classList.contains('star-icon')) {
                return;
            }
            selectEmail(email);
            // Highlight selected card
            document.querySelectorAll('.email-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
        });

        // Star toggle event
        card.querySelector('.star-icon').addEventListener('click', async (e) => {
            e.stopPropagation();
            const emailId = email.id;
            const res = await toggleActionAPI('star', emailId);
            if (res) {
                email.starred = !email.starred;
                e.target.classList.toggle('starred');
                if (state.selectedEmail && state.selectedEmail.id === emailId) {
                    state.selectedEmail.starred = email.starred;
                    updateViewerButtons();
                }
                showToast(email.starred ? "Conversation starred." : "Removed star.");
            }
        });

        // Add index delay style to slide-in animation
        card.style.animationDelay = `${index * 0.04}s`;
        elements.emailList.appendChild(card);
    });
}

// Select active email to display in detail pane
async function selectEmail(email) {
    state.selectedEmail = email;
    elements.viewerEmptyState.style.display = 'none';
    elements.viewerContent.style.display = 'flex';
    
    // Set static UI text
    elements.viewSubject.innerText = email.subject || '(No Subject)';
    elements.viewFromName.innerText = email.from_name || email.from;
    elements.viewFromEmail.innerText = `<${email.from}>`;
    elements.viewTo.innerText = email.to;
    elements.viewDate.innerText = formatDateFull(email.date);
    
    // Set letter avatar
    const letter = (email.from_name || email.from || 'U').charAt(0).toUpperCase();
    elements.senderAvatar.innerText = letter;

    updateViewerButtons();
    
    // Mark as read automatically
    if (email.unread) {
        const success = await toggleActionAPI('read', email.id, false);
        if (success) {
            email.unread = false;
            fetchEmails(); // Reload folder list counts
        }
    }

    // Load full message body content
    await fetchEmailBody(email.id);
}

function updateViewerButtons() {
    if (!state.selectedEmail) return;
    
    if (state.selectedEmail.starred) {
        elements.viewStarBtn.classList.add('starred');
        elements.viewStarBtn.innerText = '★ Starred';
    } else {
        elements.viewStarBtn.classList.remove('starred');
        elements.viewStarBtn.innerText = '☆ Star';
    }
}

// Retrieve rich/raw body content
async function fetchEmailBody(emailId) {
    elements.viewBodyMarkup.innerHTML = '<div class="loader"><div class="spinner"></div></div>';
    elements.viewBodyHtml.style.display = 'none';
    elements.viewBodyMarkup.style.display = 'block';

    try {
        const response = await apiRequest(`/api/email?id=${emailId}&folder=${state.activeFolder}`);
        if (response.ok) {
            const data = await response.json();
            // Store rich contents
            state.selectedEmail = data.email;
            renderEmailBody(data.email);
        } else {
            elements.viewBodyMarkup.innerText = 'Failed to load email contents.';
        }
    } catch (err) {
        elements.viewBodyMarkup.innerText = 'Connection error occurred.';
    }
}

// Render rich text vs html iframe
function renderEmailBody(email) {
    if (state.config.html_rich && email.is_html && email.html) {
        elements.viewBodyMarkup.style.display = 'none';
        elements.viewBodyHtml.style.display = 'block';
        
        // Write HTML content into Sandbox iframe
        const doc = elements.viewBodyHtml.contentDocument || elements.viewBodyHtml.contentWindow.document;
        doc.open();
        // Inject dynamic stylesheets within iframe to match current theme background style
        const themeStyles = window.getComputedStyle(document.body);
        const fontFam = themeStyles.getPropertyValue('font-family');
        const bgVal = themeStyles.getPropertyValue('--bg-viewer');
        const fgVal = themeStyles.getPropertyValue('--fg-primary');
        const accentVal = themeStyles.getPropertyValue('--accent');
        
        doc.write(`
            <html>
            <head>
                <style>
                    body {
                        font-family: ${fontFam};
                        font-size: 14px;
                        line-height: 1.6;
                        color: ${fgVal};
                        background-color: ${bgVal};
                        margin: 0;
                        padding: 10px;
                        word-break: break-word;
                    }
                    a {
                        color: ${accentVal};
                        text-decoration: none;
                    }
                    a:hover {
                        text-decoration: underline;
                    }
                </style>
            </head>
            <body>
                ${email.html}
            </body>
            </html>
        `);
        doc.close();
    } else {
        elements.viewBodyHtml.style.display = 'none';
        elements.viewBodyMarkup.style.display = 'block';
        elements.viewBodyMarkup.innerHTML = formatPlainBody(email.body || '(No body content)');
    }
}

// General API Save Settings
async function saveSettingsAPI(configData) {
    try {
        const response = await apiRequest('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(configData)
        });
        if (response.ok) {
            // Update local state config
            const updated = await response.json();
            state.config = { ...state.config, ...updated };
            return { success: true };
        } else {
            const text = await response.text();
            return { success: false, error: text };
        }
    } catch (err) {
        console.error("Save config error:", err);
        return { success: false, error: err.message };
    }
}

// General API action triggers (star/read/delete)
async function toggleActionAPI(action, id, val) {
    try {
        const payload = { action, id };
        if (val !== undefined) {
            payload.value = val;
        }
        const response = await apiRequest('/api/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        return response.ok;
    } catch (err) {
        console.error("API Action error:", err);
        return false;
    }
}

// Handlers for Viewer Panel actions
async function handleToggleStar() {
    if (!state.selectedEmail) return;
    const email = state.selectedEmail;
    const res = await toggleActionAPI('star', email.id);
    if (res) {
        email.starred = !email.starred;
        updateViewerButtons();
        fetchEmails(); // Reload the list card
        showToast(email.starred ? "Conversation starred." : "Removed star.");
    }
}

async function handleDeleteEmail() {
    if (!state.selectedEmail) return;
    const email = state.selectedEmail;
    const res = await toggleActionAPI('delete', email.id);
    if (res) {
        showToast("Email moved to Trash.");
        elements.viewerContent.style.display = 'none';
        elements.viewerEmptyState.style.display = 'flex';
        state.selectedEmail = null;
        fetchEmails();
    }
}

function handleReplyEmail() {
    if (!state.selectedEmail) return;
    const email = state.selectedEmail;
    openCompose();
    elements.composeTo.value = email.from;
    elements.composeSubject.value = `Re: ${email.subject.startsWith('Re: ') ? '' : 'Re: '}${email.subject}`;
    
    const replyDate = new Date(email.date).toLocaleString();
    elements.composeBody.value = `\n\nOn ${replyDate}, <${email.from}> wrote:\n> ${email.body.replace(/\n/g, '\n> ')}`;
    elements.composeBody.focus();
    elements.composeBody.setSelectionRange(0, 0); // Put cursor at top
}

// Compose Overlay operations
function openCompose() {
    elements.composeForm.reset();
    elements.composeModal.style.display = 'flex';
    elements.composeTo.focus();
}

function closeCompose() {
    elements.composeModal.style.display = 'none';
}

async function handleSendEmail(e) {
    e.preventDefault();
    const sendBtn = elements.sendComposeBtn;
    const btnText = sendBtn.querySelector('.btn-text');
    const spinner = sendBtn.querySelector('.mini-spinner');
    
    // Toggle Loading
    sendBtn.disabled = true;
    btnText.style.opacity = '0.5';
    spinner.style.display = 'inline-block';

    const payload = {
        to: elements.composeTo.value,
        subject: elements.composeSubject.value,
        body: elements.composeBody.value
    };

    try {
        const response = await apiRequest('/api/send', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            showToast(state.config.use_mock ? "Email composed successfully! (Simulated locally)" : "Email sent successfully via SMTP.");
            closeCompose();
            // Refresh sent folder if active
            if (state.activeFolder === 'sent') {
                fetchEmails();
            }
        } else {
            const errText = await response.text();
            showToast(`Error sending: ${errText}`, true);
        }
    } catch(err) {
        showToast("Server connection error during transmission.", true);
    } finally {
        sendBtn.disabled = false;
        btnText.style.opacity = '1';
        spinner.style.display = 'none';
    }
}

// Dynamic unread count loader
async function updateUnreadCounts() {
    try {
        const response = await apiRequest('/api/folders');
        if (response.ok) {
            const data = await response.json();
            const inboxCountObj = data.folders.find(f => f.name === 'inbox');
            if (inboxCountObj && inboxCountObj.unread > 0) {
                elements.inboxUnread.style.display = 'inline-block';
                elements.inboxUnread.innerText = inboxCountObj.unread;
            } else {
                elements.inboxUnread.style.display = 'none';
            }
        }
    } catch(err) {
        console.error("Count refresh failure:", err);
    }
}

// Refresh folder list & unread indicators
async function refreshApp() {
    updateUnreadCounts();
    if (state.activeFolder !== 'settings') {
        fetchEmails();
    }
}

// Helpers: escape markup to prevent injection
function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>'"]/g, 
        tag => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            "'": '&#39;',
            '"': '&quot;'
        }[tag] || tag)
    );
}

// Convert plain email text linebreaks & blockquotes to styled DOM
function formatPlainBody(text) {
    let escaped = escapeHTML(text);
    // Parse blockquotes starting with >
    escaped = escaped.split('\n').map(line => {
        if (line.trim().startsWith('&gt;')) {
            return `<blockquote>${line.trim().substring(4)}</blockquote>`;
        }
        return line;
    }).join('\n');
    
    // Replace markdown URLs
    escaped = escaped.replace(/(https?:\/\/[^\s]+)/g, '<a href="$1" target="_blank">$1</a>');
    
    return escaped.replace(/\n/g, '<br>');
}

// Utility: formatting time/dates
function formatDate(dateVal) {
    if (!dateVal) return '';
    const date = new Date(dateVal);
    const now = new Date();
    
    // If today, show time
    if (date.toDateString() === now.toDateString()) {
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    
    // If this year, show Month + Day
    if (date.getFullYear() === now.getFullYear()) {
        return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
    
    // Otherwise show short date format
    return date.toLocaleDateString([], { year: 'numeric', month: 'numeric', day: 'numeric' });
}

function formatDateFull(dateVal) {
    if (!dateVal) return '';
    const date = new Date(dateVal);
    return date.toLocaleDateString([], { 
        weekday: 'short', 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric', 
        hour: '2-digit', 
        minute: '2-digit' 
    });
}

// Toast Notification popup
function showToast(message, isError = false) {
    const toast = document.createElement('div');
    toast.className = `toast ${isError ? 'toast-error' : 'toast-success'}`;
    
    toast.innerHTML = `
        <span class="toast-message">${message}</span>
        <span class="toast-close">&times;</span>
    `;
    
    toast.querySelector('.toast-close').addEventListener('click', () => {
        toast.remove();
    });

    elements.toastContainer.appendChild(toast);

    // Auto dismiss after 4s
    setTimeout(() => {
        toast.style.animation = 'fadeOut 0.3s ease-out forwards';
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

// Unified API request helper supporting both local servers and native GUI frames
async function apiRequest(path, options = {}) {
    const method = options.method || 'GET';
    const body = options.body || '';
    
    // Dynamically resolve native functions to prevent race conditions during boot
    const nativeFunc = typeof window.goRequest === 'function' ? window.goRequest :
                       (typeof goRequest === 'function' ? goRequest : null);
    
    if (nativeFunc) {
        try {
            // Route through Go native Webview binding
            const responseText = await nativeFunc(method, path, body);
            
            // Parse response status and body from native bridge wrapper
            const responseObj = JSON.parse(responseText);
            const isOk = responseObj.status >= 200 && responseObj.status < 300;
            
            return {
                ok: isOk,
                status: responseObj.status,
                text: async () => responseObj.body,
                json: async () => JSON.parse(responseObj.body)
            };
        } catch (err) {
            console.error("Native bridge request failure:", err);
            return {
                ok: false,
                status: 500,
                text: async () => err.message
            };
        }
    } else {
        // Standard Web HTTP Server mode
        return fetch(path, options);
    }
}
