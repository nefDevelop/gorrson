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

  const themes = {
    lavender: {
      light: { primary: '#6750A4', secondary: '#625B71', tertiary: '#7D5260', container: '#EADDFF', onPrimary: '#FFFFFF', onContainer: '#21005D', surface: '#FEF7FF', onSurface: '#1D1B20', background: '#FFFFFF', variant: '#E7E0EB', outline: '#79747E' },
      dark: { primary: '#D0BCFF', secondary: '#CCC2DC', tertiary: '#EFB8C8', container: '#4F378B', onPrimary: '#381E72', onContainer: '#EADDFF', surface: '#1C1B1F', onSurface: '#E6E1E5', background: '#141218', variant: '#49454F', outline: '#938F99' }
    },
    midnight: {
      light: { primary: '#0061A4', secondary: '#535F70', tertiary: '#6B5778', container: '#D1E4FF', onPrimary: '#FFFFFF', onContainer: '#001D36', surface: '#F8F9FF', onSurface: '#191C20', background: '#FFFFFF', variant: '#DFE3EB', outline: '#73777F' },
      dark: { primary: '#9ECAFF', secondary: '#BBC7DB', tertiary: '#D6BEE4', container: '#00497D', onPrimary: '#003258', onContainer: '#D1E4FF', surface: '#111418', onSurface: '#E2E2E6', background: '#0B1015', variant: '#43474E', outline: '#8D9199' }
    },
    forest: {
      light: { primary: '#386A20', secondary: '#55624C', tertiary: '#386567', container: '#B7F397', onPrimary: '#FFFFFF', onContainer: '#042100', surface: '#F7FBF1', onSurface: '#191C19', background: '#FFFFFF', variant: '#DFE4D6', outline: '#72796F' },
      dark: { primary: '#9CD67D', secondary: '#BDCBB0', tertiary: '#A0CFD0', container: '#205107', onPrimary: '#0C3900', onContainer: '#B7F397', surface: '#1A1C19', onSurface: '#E2E3DD', background: '#111411', variant: '#424940', outline: '#8C9388' }
    },
    ruby: {
      light: { primary: '#BC1624', secondary: '#775652', tertiary: '#715B29', container: '#FFDAD6', onPrimary: '#FFFFFF', onContainer: '#410002', surface: '#FFF8F7', onSurface: '#221919', background: '#FFFFFF', variant: '#F5DDDA', outline: '#857371' },
      dark: { primary: '#FFB3AD', secondary: '#E7BDB7', tertiary: '#E0C28C', container: '#93000E', onPrimary: '#680007', onContainer: '#FFDAD6', surface: '#1A1110', onSurface: '#F1DEDC', background: '#211A19', variant: '#534341', outline: '#9F8C8A' }
    }
  };

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

  function generateTonal(hex, isDark) {
    const [h, s, l] = hexToHsl(hex);
    const primary = isDark ? hslToHex(h, s, 80) : hex;
    const secondary = hslToHex((h + 10) % 360, Math.min(s, 30), isDark ? 70 : 50);
    const tertiary = hslToHex((h + 120) % 360, Math.min(s + 10, 70), isDark ? 80 : 50);
    const neutralVariant = hslToHex(h, 12, isDark ? 25 : 90);
    if (!isDark) {
      return { primary: hex, onPrimary: '#FFFFFF', container: hslToHex(h, Math.max(s - 30, 10), 92), onContainer: hslToHex(h, s, 10), secondary, tertiary, surface: hslToHex(h, 6, 98), onSurface: '#1C1B1F', variant: neutralVariant, outline: hslToHex(h, 12, 50), background: '#FFFFFF' };
    } else {
      return { primary, onPrimary: hslToHex(h, s, 20), container: hslToHex(h, s, 30), onContainer: hslToHex(h, s, 90), secondary, tertiary, surface: hslToHex(h, 10, 12), onSurface: '#E6E1E5', variant: neutralVariant, outline: hslToHex(h, 12, 60), background: hslToHex(h, 8, 10) };
    }
  }

  function initTheme() {
    const themeId = localStorage.getItem('gorrson-theme') || 'lavender';
    const modeId = localStorage.getItem('gorrson-mode') || 'dark';
    const seed = localStorage.getItem('gorrson-custom-color') || '#6750A4';
    
    let mode = modeId === 'system' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : modeId;
    let theme = themeId === 'custom' ? generateTonal(seed, mode === 'dark') : themes[themeId][mode];
    
    const root = document.documentElement;
    root.style.setProperty('--md-primary', theme.primary);
    root.style.setProperty('--md-on-primary', theme.onPrimary);
    root.style.setProperty('--md-primary-container', theme.container);
    root.style.setProperty('--md-on-primary-container', theme.onContainer);
    root.style.setProperty('--md-secondary-container', mode === 'light' ? theme.variant : theme.container);
    root.style.setProperty('--md-on-secondary-container', theme.onSurface);
    root.style.setProperty('--md-surface', theme.surface);
    root.style.setProperty('--md-on-surface', theme.onSurface);
    root.style.setProperty('--md-surface-variant', theme.variant);
    root.style.setProperty('--md-on-surface-variant', mode === 'light' ? '#49454F' : '#CAC4D0');
    root.style.setProperty('--md-background', theme.background);
    root.style.setProperty('--md-outline', theme.outline);
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
  <main class="app-container {selectedId ? 'has-selection' : ''}">
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
