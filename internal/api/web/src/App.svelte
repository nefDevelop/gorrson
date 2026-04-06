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
  import { formatTimeAgo } from './lib/utils.js';
  import { Inbox, Star, Bookmark, Archive, Shield } from 'lucide-svelte';
  import { t } from './lib/i18n.js';
  import { onMount } from 'svelte';
  import { fetchNews, markAsRead, fetchFeeds } from './lib/api.js';

  let selectedId = null;
  let searchQuery = "";
  let isMobile = false;

  const updateMobile = () => {
    isMobile = window.innerWidth <= 800;
  };
  let articles = [];
  let feeds = [];
  let loading = true;
  let error = null;
  let isAuthenticated = false;
  let currentView = "inbox";
  let activeCategory = null;
  let activeSource = null;
  let showAddFeed = false;

  $: totalUnread = (feeds || []).reduce((acc, f) => acc + (f.unread_count || 0), 0);
  
  $: navItems = [
    { id: "inbox", name: "Inbox", badge: totalUnread, icon: Inbox },
    { id: "favorites", name: "Favorites", icon: Star },
  ];

  async function loadData() {
    try {
      loading = true;
      const [newsData, feedsData] = await Promise.all([
        fetchNews({ category: activeCategory, feed_id: activeSource, search: searchQuery }),
        fetchFeeds()
      ]);
      
      articles = (newsData || []).map(a => ({
        ...a,
        time: a.published_at ? formatTimeAgo(a.published_at) : 'Now',
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

  function hexToHsl(hex) {
    let r = parseInt(hex.slice(1, 3), 16) / 255;
    let g = parseInt(hex.slice(3, 5), 16) / 255;
    let b = parseInt(hex.slice(5, 7), 16) / 255;
    let max = Math.max(r, g, b), min = Math.min(r, g, b);
    let h, s, l = (max + min) / 2;
    if (max === min) h = s = 0;
    else {
      let d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }
      h /= 6;
    }
    return [h * 360, s * 100, l * 100];
  }

  function hslToHex(h, s, l) {
    l /= 100;
    const a = s * Math.min(l, 1 - l) / 100;
    const f = n => {
      const k = (n + h / 30) % 12;
      const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
      return Math.round(255 * color).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
  }

  function generateTonal(hex, mode) {
    const [h, s, l] = hexToHsl(hex);
    const isDark = mode === 'dark';

    // MD3 Expressive: Vibrant palettes with higher chroma
    const primary = isDark ? hslToHex(h, Math.max(s, 50), 80) : hex;
    const secondary = hslToHex((h + 25) % 360, Math.max(s - 15, 30), isDark ? 70 : 40);
    const secondaryContainer = isDark 
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 25)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90);
    const onSecondaryContainer = isDark
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 10);
    const tertiary = hslToHex((h + 80) % 360, Math.max(s, 40), isDark ? 75 : 45);

    
    // Adwaita 4 Standard Surfaces (Tinted slightly with Seed Hue)
    const adwaita = {
      dark: { 
        bg: hslToHex(h, 5, 12), 
        surface: hslToHex(h, 6, 16), 
        card: hslToHex(h, 7, 20), 
        border: hslToHex(h, 10, 24) 
      },
      light: { 
        bg: '#FFFFFF', 
        surface: hslToHex(h, 10, 96), 
        card: '#FFFFFF', 
        border: hslToHex(h, 15, 88) 
      }
    };
    const currentA = isDark ? adwaita.dark : adwaita.light;

    if (!isDark) {
      return {
        primary: hex,
        onPrimary: '#FFFFFF',
        container: hslToHex(h, Math.max(s, 70), 92),
        onContainer: hslToHex(h, s, 10),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 20, 10),
        onSurfaceVariant: hslToHex(h, 15, 40),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    } else {
      return {
        primary: primary,
        onPrimary: hslToHex(h, s, 15),
        container: hslToHex(h, Math.max(s, 60), 25),
        onContainer: hslToHex(h, Math.max(s, 40), 90),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 10, 95),
        onSurfaceVariant: hslToHex(h, 15, 75),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    }
  }

  function initTheme() {
    const modeId = localStorage.getItem('gorrson-mode') || 'dark';
    const seed = localStorage.getItem('gorrson-custom-color') || '#6750A4';
    
    let mode = modeId === 'system' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : modeId;
    let theme = generateTonal(seed, mode);
    
    const root = document.documentElement;
    root.style.setProperty('--md-primary', theme.primary);
    root.style.setProperty('--md-on-primary', theme.onPrimary);
    root.style.setProperty('--md-primary-container', theme.container);
    root.style.setProperty('--md-on-primary-container', theme.onContainer);
    root.style.setProperty('--md-secondary', theme.secondary);
    root.style.setProperty('--md-secondary-container', theme.secondaryContainer);
    root.style.setProperty('--md-on-secondary-container', theme.onSecondaryContainer);
    root.style.setProperty('--md-tertiary', theme.tertiary);
    
    root.style.setProperty('--md-surface', theme.surface);
    root.style.setProperty('--md-on-surface', theme.onSurface);
    root.style.setProperty('--md-surface-variant', theme.variant);
    root.style.setProperty('--md-on-surface-variant', theme.onSurfaceVariant);
    
    root.style.setProperty('--md-background', theme.background);
    root.style.setProperty('--md-outline', theme.outline);
    
    // UI Effects
    root.style.setProperty('--shadow-color', mode === 'light' ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.3)');
  }

  onMount(() => {
    updateMobile();
    window.addEventListener('resize', updateMobile);
    initTheme();
    
    if (localStorage.getItem('token')) {
      isAuthenticated = true;
      loadData();
    }
    return () => window.removeEventListener('resize', updateMobile);
  });

  function handleViewChange(viewId) {
    if (viewId.startsWith('cat:')) {
      activeCategory = viewId.replace('cat:', '');
      activeSource = null;
      currentView = 'inbox';
    } else if (viewId.startsWith('src:')) {
      activeSource = viewId.replace('src:', '');
      activeCategory = null;
      currentView = 'inbox';
    } else {
      activeCategory = null;
      activeSource = null;
      currentView = viewId;
    }
    loadData();
  }

  // Real-time search
  $: if (isAuthenticated && searchQuery !== undefined) {
    const timer = setTimeout(loadData, 300);
  }

  $: selectedArticle = articles.find(n => n.id === selectedId);

  $: categories = Array.from(new Set((feeds || []).map(f => f.category || 'General'))).map(catName => ({
    id: catName.toLowerCase(),
    name: catName,
    count: feeds.filter(f => (f.category || 'General') === catName).reduce((acc, f) => acc + f.unread_count, 0)
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
  <main class="app-container {selectedId ? 'has-selection' : ''}">
    <Sidebar 
      {navItems} 
      {categories} 
      {feeds}
      activeView={activeCategory ? 'cat:'+activeCategory : (activeSource ? 'src:'+activeSource : currentView)} 
      onViewChange={handleViewChange} 
      onAddFeed={handleAddFeed}
      onRefresh={loadData}
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
        {#if !isMobile || (isMobile && !selectedId)}
          <FeedList 
            {articles} 
            {selectedId} 
            onSelect={handleSelect} 
          />
        {/if}
        
        {#if !isMobile || (isMobile && selectedId)}
          <PostViewer 
            article={selectedArticle} 
            onBack={() => selectedId = null}
          />
        {/if}
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
