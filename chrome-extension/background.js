// Background service worker for GitHub Git Notes Viewer
// Handles API calls to bypass CORS restrictions and monitors tabs for notes

// Cache for API responses (5 minutes TTL)
const cache = new Map();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

// Track tab states
const tabStates = new Map();

// Listen for tab updates
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete' && tab.url) {
    checkTabForNotes(tabId, tab.url);
  }
});

// Listen for tab activation
chrome.tabs.onActivated.addListener(async (activeInfo) => {
  const tab = await chrome.tabs.get(activeInfo.tabId);
  if (tab.url) {
    checkTabForNotes(activeInfo.tabId, tab.url);
  }
});

// Check if a tab is on a GitHub commit page and has notes
async function checkTabForNotes(tabId, url) {
  const commitMatch = url.match(/^https:\/\/github\.com\/([^\/]+)\/([^\/]+)\/commit\/([0-9a-f]+)/i);
  
  if (!commitMatch) {
    // Not a commit page, clear badge
    chrome.action.setBadgeText({ tabId, text: '' });
    tabStates.delete(tabId);
    return;
  }
  
  const [, owner, repo, commitSha] = commitMatch;
  const cacheKey = `${owner}/${repo}/${commitSha}`;
  
  // Check if we already know about this tab
  const tabState = tabStates.get(tabId);
  if (tabState && tabState.cacheKey === cacheKey && tabState.checked) {
    return;
  }
  
  try {
    // Check cache first
    const cached = cache.get(cacheKey);
    let noteCount = 0;
    
    if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
      noteCount = Object.keys(cached.data).length;
    } else {
      // Fetch notes to check if any exist
      const notesRefs = await fetchNotesRefs(owner, repo);
      
      for (const ref of notesRefs) {
        const refName = ref.replace('refs/notes/', '');
        const hasNotes = await checkCommitHasNotes(owner, repo, commitSha, refName);
        if (hasNotes) {
          noteCount++;
        }
      }
    }
    
    // Update badge
    if (noteCount > 0) {
      chrome.action.setBadgeText({ tabId, text: noteCount.toString() });
      chrome.action.setBadgeBackgroundColor({ tabId, color: '#0366d6' });
    } else {
      chrome.action.setBadgeText({ tabId, text: '' });
    }
    
    // Save state
    tabStates.set(tabId, {
      cacheKey,
      noteCount,
      checked: true
    });
    
  } catch (error) {
    console.error('Error checking for notes:', error);
    // Don't show badge on error
    chrome.action.setBadgeText({ tabId, text: '' });
  }
}

// Quick check if a commit has notes without fetching content
async function checkCommitHasNotes(owner, repo, commitSha, notesRef) {
  try {
    const refResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/refs/notes/${notesRef}`
    );
    
    if (!refResponse.ok) {
      return false;
    }
    
    const refData = await refResponse.json();
    const notesSha = refData.object.sha;
    
    // Get the tree
    const commitResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/commits/${notesSha}`
    );
    
    if (!commitResponse.ok) {
      return false;
    }
    
    const commitData = await commitResponse.json();
    const treeSha = commitData.tree.sha;
    
    const treeResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/trees/${treeSha}`
    );
    
    if (!treeResponse.ok) {
      return false;
    }
    
    const treeData = await treeResponse.json();
    
    // Check if this commit exists in the tree
    return treeData.tree.some(item => 
      item.path.startsWith(commitSha) || commitSha.startsWith(item.path.substring(0, 7))
    );
    
  } catch (error) {
    console.error(`Error checking notes for ref ${notesRef}:`, error);
    return false;
  }
}

// Listen for messages from content script
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'fetchGitNotes') {
    handleFetchGitNotes(request, sendResponse);
    return true; // Will respond asynchronously
  }
});

async function handleFetchGitNotes(request, sendResponse) {
  const { owner, repo, commitSha } = request;
  const cacheKey = `${owner}/${repo}/${commitSha}`;
  
  // Check cache first
  const cached = cache.get(cacheKey);
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    sendResponse({ success: true, data: cached.data });
    return;
  }
  
  try {
    // Fetch all available notes refs
    const notesRefs = await fetchNotesRefs(owner, repo);
    const allNotes = {};
    
    for (const ref of notesRefs) {
      const refName = ref.replace('refs/notes/', '');
      const notes = await fetchNotesForCommit(owner, repo, commitSha, refName);
      if (notes) {
        allNotes[refName] = notes;
      }
    }
    
    // Cache the result
    cache.set(cacheKey, {
      data: allNotes,
      timestamp: Date.now()
    });
    
    sendResponse({ success: true, data: allNotes });
  } catch (error) {
    console.error('Error fetching git notes:', error);
    sendResponse({ 
      success: false, 
      error: error.message,
      rateLimitRemaining: error.rateLimitRemaining
    });
  }
}

async function fetchNotesRefs(owner, repo) {
  const response = await fetch(`https://api.github.com/repos/${owner}/${repo}/git/refs`);
  
  if (!response.ok) {
    throw createError(response);
  }
  
  const refs = await response.json();
  return refs
    .filter(ref => ref.ref.startsWith('refs/notes/'))
    .map(ref => ref.ref);
}

async function fetchNotesForCommit(owner, repo, commitSha, notesRef) {
  try {
    // Step 1: Get the notes reference
    const refResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/refs/notes/${notesRef}`
    );
    
    if (!refResponse.ok) {
      if (refResponse.status === 404) {
        return null; // No notes in this ref
      }
      throw createError(refResponse);
    }
    
    const refData = await refResponse.json();
    const notesSha = refData.object.sha;
    
    // Step 2: Get the commit object
    const commitResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/commits/${notesSha}`
    );
    
    if (!commitResponse.ok) {
      throw createError(commitResponse);
    }
    
    const commitData = await commitResponse.json();
    const treeSha = commitData.tree.sha;
    
    // Step 3: Get the tree
    const treeResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/trees/${treeSha}`
    );
    
    if (!treeResponse.ok) {
      throw createError(treeResponse);
    }
    
    const treeData = await treeResponse.json();
    
    // Find the blob for this commit (handle short SHAs)
    const blob = treeData.tree.find(item => 
      item.path.startsWith(commitSha) || commitSha.startsWith(item.path.substring(0, 7))
    );
    
    if (!blob) {
      return null; // No notes for this commit
    }
    
    // Step 4: Get the blob content
    const blobResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/blobs/${blob.sha}`
    );
    
    if (!blobResponse.ok) {
      throw createError(blobResponse);
    }
    
    const blobData = await blobResponse.json();
    
    // Decode base64 content
    const content = atob(blobData.content);
    
    // Try to parse as JSON
    try {
      return JSON.parse(content);
    } catch {
      // Return as plain text if not JSON
      return { content };
    }
    
  } catch (error) {
    console.error(`Error fetching notes for ref ${notesRef}:`, error);
    throw error;
  }
}

function createError(response) {
  const error = new Error(`GitHub API error: ${response.status} ${response.statusText}`);
  error.status = response.status;
  error.rateLimitRemaining = response.headers.get('X-RateLimit-Remaining');
  return error;
}