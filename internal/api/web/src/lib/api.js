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
