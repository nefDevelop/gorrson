<script>
  import './app.css';
  import Sidebar from './lib/Sidebar.svelte';
  import FeedList from './lib/FeedList.svelte';
  import PostViewer from './lib/PostViewer.svelte';
  import CommandPalette from './lib/CommandPalette.svelte';
  import Login from './lib/Login.svelte';
  import Filters from './lib/Filters.svelte';
  import Settings from './lib/Settings.svelte';
  import AddFeed from './lib/AddFeed.svelte';
  import Toaster from './lib/Toaster.svelte';
  import { success as toastSuccess, error as toastError } from './lib/toast.js';
  import { Inbox, Star, Bookmark, Archive, Shield } from 'lucide-svelte';
  import { t } from './lib/i18n.js';
  import { onMount } from 'svelte';
  import { fetchNews, markAsRead, fetchFeeds } from './lib/api.js';

  let selectedId = null;
  let searchQuery = "";
  let articles = [];
  let feeds = [];
  let loading = true;
  let error = null;
  let isAuthenticated = false;
  let currentView = "inbox";
  let activeCategory = null;
  let showAddFeed = false;

  const navItems = [
    { id: "inbox", name: "Inbox", badge: 0, icon: Inbox },
    { id: "favorites", name: "Favorites", icon: Star },
  ];

  async function loadData() {
    try {
      loading = true;
      const [newsData, feedsData] = await Promise.all([
        fetchNews({ category: activeCategory, search: searchQuery }),
        fetchFeeds()
      ]);
      
      articles = (newsData || []).map(a => ({
        ...a,
        time: a.published_at ? new Date(a.published_at).toLocaleDateString() : 'Now',
        source: (feedsData || []).find(f => f.id === a.feed_id)?.title || 'RSS',
        category: (feedsData || []).find(f => f.id === a.feed_id)?.category || 'General'
      }));

      feeds = feedsData || [];
      if (articles.length > 0 && !selectedId) selectedId = articles[0].id;
    } catch (e) {
      error = e.message;
      if (e.message === 'Unauthorized') isAuthenticated = false;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    if (localStorage.getItem('token')) {
      isAuthenticated = true;
      loadData();
    }
  });

  function handleViewChange(viewId) {
    if (viewId.startsWith('cat:')) {
      activeCategory = viewId.replace('cat:', '');
      currentView = 'inbox';
    } else {
      activeCategory = null;
      currentView = viewId;
    }
    loadData();
  }

  // Real-time search
  $: if (isAuthenticated && searchQuery !== undefined) {
    const timer = setTimeout(loadData, 300);
  }

  $: selectedArticle = articles.find(n => n.id === selectedId);

  $: categories = Array.from(new Set((feeds || []).map(f => f.category || 'General'))).map(catId => ({
    id: catId.toLowerCase(),
    name: catId,
    count: articles.filter(a => a.category === catId).length
  }));

  async function handleSelect(id) {
    selectedId = id;
    await markAsRead(id);
  }

  function handleLoginSuccess() {
    isAuthenticated = true;
    loadData();
  }

  function handleLogout() {
    localStorage.removeItem('token');
    isAuthenticated = false;
  }

  function handleAddFeed() {
    showAddFeed = true;
  }

  function handleAddFeedSave(feedTitle) {
    showAddFeed = false;
    toastSuccess(`Feed "${feedTitle || 'nuevo feed'}" añadido correctamente ✓`);
    loadData();
  }

  function handleAddFeedCancel() {
    showAddFeed = false;
  }
</script>

{#if !isAuthenticated}
  <Login onLoginSuccess={handleLoginSuccess} />
{:else}
  <main class="app-container">
    <Sidebar 
      {navItems} 
      {categories} 
      activeView={activeCategory ? 'cat:'+activeCategory : currentView} 
      onViewChange={handleViewChange} 
      onAddFeed={handleAddFeed}
    />
    
    {#if loading && articles.length === 0}
      <div class="loading-overlay">
        <div class="spinner"></div>
        <p>Loading Gorrson...</p>
      </div>
    {:else if error}
      <div class="error-view">
        <p>{error}</p>
        <button on:click={loadData}>Retry</button>
      </div>
    {:else}
      {#if currentView === 'inbox'}
        <FeedList 
          {articles} 
          {selectedId} 
          onSelect={handleSelect} 
        />
        <PostViewer article={selectedArticle} />
      {:else if currentView === 'filters'}
        <Filters />
      {:else if currentView === 'settings'}
        <Settings {feeds} onLogout={handleLogout} />
      {/if}
    {/if}

    <CommandPalette bind:searchQuery placeholder={$t('search_placeholder')} />

    {#if showAddFeed}
      <AddFeed onSave={handleAddFeedSave} onCancel={handleAddFeedCancel} />
    {/if}
  </main>
  <Toaster />
{/if}

<style>
  .app-container {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }
  .loading-overlay, .error-view {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    background: var(--md-background);
    color: var(--md-on-surface-variant);
    font: var(--m3-body-large);
  }
  .spinner {
    width: 40px;
    height: 40px;
    border: 4px solid var(--md-surface-variant);
    border-top: 4px solid var(--md-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
