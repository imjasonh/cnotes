// Popup script for GitHub Git Notes Viewer

document.addEventListener('DOMContentLoaded', async () => {
  const statusEl = document.getElementById('status');
  const contentEl = document.getElementById('content');
  
  try {
    // Get the current tab
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    
    // Check if we're on a GitHub commit page
    const commitMatch = tab.url.match(/^https:\/\/github\.com\/([^\/]+)\/([^\/]+)\/commit\/([0-9a-f]+)/i);
    
    if (!commitMatch) {
      showMessage('Not a GitHub commit page', 'warning');
      return;
    }
    
    const [, owner, repo, commitSha] = commitMatch;
    
    // Update status
    statusEl.innerHTML = `
      <span class="repo-info">${owner}/${repo}</span>
      <span class="commit-sha">${commitSha.substring(0, 7)}</span>
    `;
    
    // Fetch notes
    const response = await new Promise((resolve) => {
      chrome.runtime.sendMessage({
        action: 'fetchGitNotes',
        owner,
        repo,
        commitSha
      }, resolve);
    });
    
    if (!response.success) {
      showError(response.error, response.rateLimitRemaining);
      return;
    }
    
    displayNotes(response.data);
    
  } catch (error) {
    console.error('Error:', error);
    showError(error.message);
  }
});

function displayNotes(notes) {
  const contentEl = document.getElementById('content');
  const noteRefs = Object.keys(notes);
  
  if (noteRefs.length === 0) {
    showMessage('No git notes found for this commit', 'info');
    return;
  }
  
  let html = '';
  
  if (noteRefs.length === 1) {
    // Single ref - show directly
    const refName = noteRefs[0];
    html = `
      <div class="note-ref">
        <h2>${refName}</h2>
        ${formatNote(notes[refName])}
      </div>
    `;
  } else {
    // Multiple refs - create tabs
    html = '<div class="tabs">';
    html += '<div class="tab-buttons">';
    
    noteRefs.forEach((ref, index) => {
      const isActive = index === 0;
      html += `
        <button class="tab-button ${isActive ? 'active' : ''}" 
                data-ref="${ref}">
          ${ref}
        </button>
      `;
    });
    
    html += '</div>';
    html += '<div class="tab-content">';
    
    noteRefs.forEach((ref, index) => {
      const isActive = index === 0;
      html += `
        <div class="tab-panel ${isActive ? 'active' : ''}" data-ref="${ref}">
          ${formatNote(notes[ref])}
        </div>
      `;
    });
    
    html += '</div></div>';
  }
  
  contentEl.innerHTML = html;
  
  // Set up tab switching
  if (noteRefs.length > 1) {
    setupTabs();
  }
}

function formatNote(noteData) {
  // Check if it's a cnotes-style note
  if (noteData.session_id && noteData.conversation_excerpt) {
    return formatCnotesNote(noteData);
  }
  
  // Check if it's JSON
  if (typeof noteData === 'object' && !noteData.content) {
    return `<pre class="json-content">${JSON.stringify(noteData, null, 2)}</pre>`;
  }
  
  // Plain text
  const content = noteData.content || noteData;
  return `<pre class="text-content">${escapeHtml(content)}</pre>`;
}

function formatCnotesNote(data) {
  let html = '<div class="cnotes-content">';
  
  // Metadata
  html += '<div class="cnotes-meta">';
  html += `<span class="badge">Session: ${data.session_id.substring(0, 8)}...</span>`;
  if (data.claude_version) {
    html += `<span class="badge badge-primary">${data.claude_version}</span>`;
  }
  html += `<span class="timestamp">${new Date(data.timestamp).toLocaleString()}</span>`;
  html += '</div>';
  
  // Tools used
  if (data.tools_used && data.tools_used.length > 0) {
    html += '<div class="tools-section">';
    html += '<strong>Tools used:</strong> ';
    data.tools_used.forEach(tool => {
      html += `<span class="badge badge-secondary">${tool}</span>`;
    });
    html += '</div>';
  }
  
  // Conversation excerpt
  if (data.conversation_excerpt) {
    html += '<div class="conversation-section">';
    html += '<strong>Conversation:</strong>';
    html += '<div class="conversation-excerpt">';
    
    let formatted = escapeHtml(data.conversation_excerpt);
    formatted = formatted.replace(/👤 User:|🧑 User:/g, '<strong class="user">$&</strong>');
    formatted = formatted.replace(/🤖 Claude:/g, '<strong class="assistant">$&</strong>');
    formatted = formatted.replace(/Tool \(([^)]+)\):/g, '<em class="tool">Tool ($1):</em>');
    formatted = formatted.replace(/\n/g, '<br>');
    
    html += formatted;
    html += '</div></div>';
  }
  
  html += '</div>';
  return html;
}

function setupTabs() {
  const buttons = document.querySelectorAll('.tab-button');
  const panels = document.querySelectorAll('.tab-panel');
  
  buttons.forEach(button => {
    button.addEventListener('click', () => {
      const ref = button.dataset.ref;
      
      // Update buttons
      buttons.forEach(btn => {
        btn.classList.toggle('active', btn === button);
      });
      
      // Update panels
      panels.forEach(panel => {
        panel.classList.toggle('active', panel.dataset.ref === ref);
      });
    });
  });
}

function showMessage(message, type = 'info') {
  const contentEl = document.getElementById('content');
  contentEl.innerHTML = `
    <div class="message message-${type}">
      <p>${message}</p>
    </div>
  `;
}

function showError(error, rateLimitRemaining) {
  let message = error;
  
  if (error.includes('403') || error.includes('rate limit')) {
    message = 'GitHub API rate limit exceeded.';
    if (rateLimitRemaining !== undefined) {
      message += ` (${rateLimitRemaining} requests remaining)`;
    }
    message += '<br><small>Limit: 60 requests/hour for unauthenticated access</small>';
  }
  
  const contentEl = document.getElementById('content');
  contentEl.innerHTML = `
    <div class="message message-error">
      <p>${message}</p>
    </div>
  `;
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}