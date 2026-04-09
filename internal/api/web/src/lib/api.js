export async function fetchNews(params = {}) {
  // Remove null/undefined/empty values to avoid spurious filters like category=null
  const clean = Object.fromEntries(
    Object.entries(params).filter(([, v]) => v != null && v !== '' && v !== 'null')
  );
  const query = new URLSearchParams(clean).toString();
  const res = await fetch(`/news?${query}`, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch news');
  return res.json();
}

export async function fetchFeeds() {
  const res = await fetch('/feeds', {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch feeds');
  return res.json();
}

export async function markAsRead(id) {
  const res = await fetch(`/news/${id}/read`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  return res.ok;
}

export async function markAllRead(params = {}) {
  const clean = Object.fromEntries(
    Object.entries(params).filter(([, v]) => v != null && v !== '' && v !== 'null')
  );
  const query = new URLSearchParams(clean).toString();
  const res = await fetch(`/news/read-all?${query}`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  return res.ok;
}

export async function toggleFavorite(id, favorite = true) {
  const res = await fetch(`/news/${id}/favorite?favorite=${favorite}`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  return res.ok;
}

export async function fetchFilters() {
  const res = await fetch('/filters', {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch filters');
  return res.json();
}

export async function addFilter(filter) {
  const res = await fetch('/filters', {
    method: 'POST',
    body: JSON.stringify(filter),
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  return res.ok;
}

export async function deleteFilter(id) {
  const res = await fetch(`/filters/${id}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  return res.ok;
}

export async function login(username, password) {
  const res = await fetch('/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  });
  if (!res.ok) throw new Error('Invalid credentials');
  const data = await res.json();
  localStorage.setItem('token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  localStorage.setItem('username', username);
  return data;
}

export async function addFeed(url, category = '') {
  const res = await fetch('/feeds', {
    method: 'POST',
    body: JSON.stringify({ url, category }),
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
      'Content-Type': 'application/json'
    }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || 'Failed to add feed');
  }
  try {
    return await res.json();
  } catch {
    return {};
  }
}

export async function updateFeed(id, data) {
  const res = await fetch(`/feeds/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
      'Content-Type': 'application/json'
    }
  });
  if (!res.ok) throw new Error('Failed to update feed');
  return res.json();
}

export async function deleteFeed(id) {
  const res = await fetch(`/feeds/${id}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (!res.ok) throw new Error('Failed to delete feed');
}
export async function fetchFullContent(id) {
  const res = await fetch(`/news/${id}/content`, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch full content');
  return res.text();
}

export async function fetchAPIKeys() {
  const res = await fetch('/api-keys', {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch API keys');
  return res.json();
}

export async function createAPIKey(name) {
  const res = await fetch('/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
      'Content-Type': 'application/json'
    }
  });
  if (!res.ok) throw new Error('Failed to create API key');
  return res.json();
}

export async function deleteAPIKey(id) {
  const res = await fetch(`/api-keys/${id}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (!res.ok) throw new Error('Failed to delete API key');
}

export async function fetchStats() {
  const res = await fetch('/stats', {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`
    }
  });
  if (res.status === 401) throw new Error('Unauthorized');
  if (!res.ok) throw new Error('Failed to fetch stats');
  return res.json();
}
