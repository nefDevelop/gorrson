<script>
  import './app.css';
  import Sidebar from './lib/Sidebar.svelte';
  import FeedList from './lib/FeedList.svelte';
  import PostViewer from './lib/PostViewer.svelte';
  import Login from './lib/Login.svelte';
  import Filters from './lib/Filters.svelte';
  import Settings from './lib/Settings.svelte';
  import AddFeed from './lib/AddFeed.svelte';
  import Insights from './lib/Insights.svelte';
  import Toaster from './lib/Toaster.svelte';
  import MobileNav from './lib/MobileNav.svelte';
  import { success as toastSuccess, error as toastError } from './lib/toast.js';
  import { formatTimeAgo } from './lib/utils.js';
  import { Inbox, Star, Bookmark, Archive, Shield, BarChart2 } from 'lucide-svelte';
  import { t } from './lib/i18n.js';
  import { onMount } from 'svelte';
  import { fetchNews, markAsRead, markAllRead, fetchFeeds, toggleFavorite } from './lib/api.js';

  let selectedId = null;
  let searchQuery = "";
  let isMobile = typeof window !== 'undefined' ? window.innerWidth <= 768 : false;
  let isSidebarMini = typeof window !== 'undefined' ? window.innerWidth <= 1000 : false;
  
  let offset = 0;
  const LIMIT = 50;
  let allLoaded = false;
  let loadingMore = false;
  let scrollRead = localStorage.getItem('gorrson-scroll-read') === 'true';
  let unreadOnly = false;
  let sortMode = "newest"; // "newest", "oldest"
  let cardStyle = localStorage.getItem('gorrson-card-style') || 'default';

  $: localStorage.setItem('gorrson-scroll-read', scrollRead.toString());
  $: localStorage.setItem('gorrson-card-style', cardStyle);

  const STOP_WORDS = new Set(['de', 'a', 'el', 'la', 'en', 'y', 'que', 'os', 'un', 'una', 'con', 'para', 'por', 'lo', 'las', 'los', 'su', 'al', 'se', 'del', 'the', 'and', 'with', 'that', 'this', 'from']);

  const updateMobile = () => {
    isMobile = window.innerWidth <= 768;
    isSidebarMini = window.innerWidth <= 1000;
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
  let isSourcesDrawerOpen = false;
  let lastSyncDate = new Date();
  let now = new Date();

  // Update relative times every 30 seconds
  onMount(() => {
    const interval = setInterval(() => {
      now = new Date();
    }, 30000);

    // Background polling every 5 minutes for new articles (and notifications)
    const pollInterval = setInterval(() => {
      if (isAuthenticated && !loading && !loadingMore && !searchQuery) {
        loadData();
      }
    }, 5 * 60 * 1000);

    return () => {
      clearInterval(interval);
      clearInterval(pollInterval);
    };
  });

  $: lastSyncFormatted = formatTimeAgo(lastSyncDate);

  $: totalUnread = (feeds || []).reduce((acc, f) => acc + (f.unread_count || 0), 0);
  
  $: navItems = [
    { id: "inbox", name: "Inbox", badge: totalUnread, icon: Inbox },
    { id: "favorites", name: "Favorites", icon: Star },
    { id: "insights", name: "Insights", icon: BarChart2 },
  ];

  async function loadData(append = false) {
    try {
      if (!append) {
        loading = true;
        offset = 0;
        allLoaded = false;
      } else {
        loadingMore = true;
      }

      const params = { 
        category: activeCategory, 
        feed_id: activeSource, 
        sort: currentView === 'favorites' ? 'favorite' : sortMode,
        unread: unreadOnly ? 'true' : undefined,
        search: searchQuery,
        limit: LIMIT,
        offset: offset
      };
      
      const [newsData, feedsData] = await Promise.all([
        fetchNews(params),
        fetchFeeds()
      ]);
      
      const newArticles = (newsData || []).map(a => ({
        ...a,
        time: a.published_at ? formatTimeAgo(a.published_at) : 'Now',
        source: (feedsData || []).find(f => f.id === a.feed_id)?.title || 'RSS',
        category: (feedsData || []).find(f => f.id === a.feed_id)?.category || 'General'
      }));

      // Notification logic
      if (!append) {
        checkNotifications(newArticles);
      }

      if (append) {
        articles = [...articles, ...newArticles];
      } else {
        articles = newArticles;
      }
      
      if (newArticles.length < LIMIT) {
        allLoaded = true;
      }
      
      offset += newArticles.length;
      feeds = feedsData || [];
      
      // Auto-selection only for full desktop
      if (articles.length > 0 && !selectedId && !append && !isMobile && !isSidebarMini) {
        selectedId = articles[0].id;
      }
      lastSyncDate = new Date();
    } catch (e) {
      error = e.message;
      if (e.message === 'Unauthorized') isAuthenticated = false;
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  let notifiedIds = new Set();
  async function checkNotifications(newEntries) {
    if (typeof window === 'undefined' || !("Notification" in window)) return;
    
    if (Notification.permission === "default") {
      await Notification.requestPermission();
    }

    if (Notification.permission !== "granted") return;

    const toNotify = (newEntries || []).filter(e => 
      e.matched_action === 'NOTIFY' && 
      !e.is_read && 
      !notifiedIds.has(e.id)
    );

    if (toNotify.length === 0) return;

    // Mark as notified to avoid repetitions in this session
    toNotify.forEach(e => notifiedIds.add(e.id));

    if (toNotify.length > 3) {
      new Notification(`Gorrson: ${toNotify.length} news items`, {
        body: $t('inbox') + ": " + toNotify.map(e => e.title).slice(0, 3).join(", ") + "...",
        icon: "/favicon.png"
      });
    } else {
      toNotify.forEach(item => {
        new Notification(item.title, {
          body: `${item.source} (${item.category})`,
          icon: "/favicon.png",
          tag: item.id // Tag prevents duplicate notifications for the same ID
        });
      });
    }
  }

  function handleLoadMore() {
    if (!allLoaded && !loadingMore && !loading) {
      loadData(true);
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
        onContainer: hslToHex(h, Math.max(s, 40), 80),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 10, 85),
        onSurfaceVariant: hslToHex(h, 15, 65),
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
    selectedId = null;
    let shouldLoad = true;

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
      if (viewId === 'settings' || viewId === 'add-feed' || viewId === 'insights') {
        shouldLoad = false;
        loading = false;
      }
    }
    
    isSourcesDrawerOpen = false;
    if (shouldLoad) {
      articles = []; // Reset list only for news views
      loadData();
    }
  }

  // Real-time search - disabled network trigger to use client-side filtering
  /*
  $: if (isAuthenticated && searchQuery !== undefined) {
    const timer = setTimeout(loadData, 300);
  }
  */

  $: if (isAuthenticated && (unreadOnly !== undefined || sortMode)) {
    loadData();
  }

  // Pre-filter search logic (client side)
  $: filteredArticles = (() => {
    if (!searchQuery || searchQuery.trim().length === 0) return articles;
    
    const tokens = searchQuery.toLowerCase()
      .split(/\s+/)
      .filter(word => word.length > 2 && !STOP_WORDS.has(word));
    
    if (tokens.length === 0 && searchQuery.trim().length > 0) {
      // If we have query but all are stop words, just do a literal match of the whole query
      return articles.filter(a => a.title.toLowerCase().includes(searchQuery.toLowerCase()));
    }
    
    if (tokens.length === 0) return articles;

    return articles.filter(article => {
      const searchContent = `${article.title} ${article.summary || ''} ${article.content || ''}`.toLowerCase();
      return tokens.every(token => searchContent.includes(token));
    });
  })();

  $: selectedArticle = articles.find(n => n.id === selectedId);

  $: categories = Array.from(new Set((feeds || []).map(f => f.category || 'General'))).map(catName => ({
    id: catName.toLowerCase(),
    name: catName,
    count: feeds.filter(f => (f.category || 'General') === catName).reduce((acc, f) => acc + f.unread_count, 0)
  }));

  async function handleMarkAsRead(id) {
    if (articles.find(a => a.id === id)?.is_read) return;
    await markAsRead(id);
    articles = articles.map(a => a.id === id ? {...a, is_read: true} : a);
  }

  async function handleSelect(id) {
    selectedId = id;
    handleMarkAsRead(id);
  }

  async function handleMarkAllRead() {
    try {
      const params = {
        category: activeCategory,
        feed_id: activeSource,
        search: searchQuery,
        limit: articles.length // Only mark what's loaded in the UI
      };
      const unreadCount = articles.filter(a => !a.is_read).length;
      await markAllRead(params);
      
      articles = articles.map(a => ({...a, is_read: true}));
      toastSuccess(`Marked ${unreadCount} items as read`);
    } catch (e) {
      toastError('Failed to mark as read');
    }
  }

  async function handleToggleFavorite(id) {
    const article = articles.find(a => a.id === id);
    if (!article) return;
    
    const newState = !article.is_favorite;
    await toggleFavorite(id, newState);
    
    // Update local state across all views
    articles = articles.map(a => a.id === id ? {...a, is_favorite: newState} : a);
    
    // If we're in favorites view, remove it if un-starred
    if (currentView === 'favorites' && !newState) {
       articles = articles.filter(a => a.id !== id);
       if (selectedId === id) selectedId = null;
    }
  }

  async function handleShare(article) {
    if (navigator.share) {
      try {
        await navigator.share({
          title: article.title,
          url: article.url
        });
      } catch (e) {
        await copyToClipboard(article.url);
      }
    } else {
      await copyToClipboard(article.url);
    }
  }

  async function copyToClipboard(url) {
    await navigator.clipboard.writeText(url);
    toastSuccess($t('link_copied'));
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

  let isFeedExpanded = false;
  let fullscreenReader = false;
  const toggleFeedExpand = () => isFeedExpanded = !isFeedExpanded;
  const toggleFullscreen = () => fullscreenReader = !fullscreenReader;

  // Auto-collapse feed list if user shrinks the browser
  $: if (isMobile && isFeedExpanded) {
    isFeedExpanded = false;
  }
</script>

{#if !isAuthenticated}
  <Login onLoginSuccess={handleLoginSuccess} />
{:else}
  <main class="app-container {selectedId ? 'has-selection' : ''} {isMobile ? 'mobile-mode' : ''}">
    <div class="nav-panels" class:collapsed={fullscreenReader} class:full-width={currentView === 'settings' || currentView === 'filters' || currentView === 'add-feed' || currentView === 'insights'}>
      {#if !isMobile || isSourcesDrawerOpen}
        <div 
          class="sidebar-wrapper" 
          class:mobile-drawer={isMobile} 
          class:open={isSourcesDrawerOpen} 
          on:click|self={() => isSourcesDrawerOpen = false}
          on:keydown={(e) => e.key === 'Escape' && (isSourcesDrawerOpen = false)}
          role="button"
          tabindex={isSourcesDrawerOpen ? 0 : -1}
          aria-label="Close menu"
        >
          <Sidebar 
            {navItems} 
            {categories} 
            {feeds}
            {isMobile}
            lastSync={lastSyncFormatted}
            isMini={isSidebarMini}
            activeView={activeCategory ? 'cat:'+activeCategory : (activeSource ? 'src:'+activeSource : currentView)} 
            onViewChange={handleViewChange} 
            onAddFeed={handleAddFeed}
            onRefresh={loadData}
          />
        </div>
      {/if}
      
      {#if loading && articles.length === 0}
        <div class="loading-overlay">
          <div class="spinner"></div>
          <p>Loading Gorrson...</p>
        </div>
      {:else if error}
        <div class="error-view">
          <p>{error}</p>
          <button on:click={() => loadData()}>Retry</button>
        </div>
      {:else}
        {#if currentView === 'inbox' || currentView === 'favorites'}
          {#if (!isSidebarMini && !isMobile) || (isSidebarMini && !selectedId) || (isMobile && !selectedId)}
            <FeedList 
              bind:searchQuery
              articles={filteredArticles} 
              selectedId={selectedId} 
              isExpanded={isFeedExpanded}
              onToggleExpand={toggleFeedExpand}
              onSelect={handleSelect} 
              onMarkAsRead={handleMarkAsRead}
              onToggleFavorite={handleToggleFavorite}
              onShare={handleShare}
              onMarkAllRead={handleMarkAllRead}
              onLoadMore={handleLoadMore}
              {loadingMore}
              {allLoaded}
              {scrollRead}
              {cardStyle}
              bind:unreadOnly
              bind:sortMode
            />
          {/if}
        {:else if currentView === 'filters'}
          <Filters />
        {:else if currentView === 'insights'}
          <Insights />
        {:else if currentView === 'settings'}
          <Settings 
            {feeds} 
            {isMobile}
            onLogout={handleLogout} 
            username={localStorage.getItem('username')}
            bind:scrollRead
            bind:cardStyle
          />
        {/if}
      {/if}
    </div>

    {#if (currentView === 'inbox' || currentView === 'favorites') && filteredArticles.length > 0 && (!isMobile || (isMobile && selectedId))}
      <PostViewer 
        article={selectedArticle} 
        onBack={() => selectedId = null}
        isFullscreen={fullscreenReader}
        onToggleFullscreen={toggleFullscreen}
      />
    {/if}

    {#if showAddFeed}
      <AddFeed onSave={handleAddFeedSave} onCancel={handleAddFeedCancel} />
    {/if}

    {#if isMobile}
      <MobileNav 
        activeView={currentView} 
        totalUnread={totalUnread}
        onNavClick={handleViewChange}
        onToggleSources={() => isSourcesDrawerOpen = !isSourcesDrawerOpen}
      />
    {/if}
  </main>
  <Toaster />
{/if}

<style>
  .app-container {
    display: flex;
    height: 100vh;
    overflow: hidden;
    background: var(--md-background);
  }

  .nav-panels {
    display: flex;
    height: 100%;
    transition: all 0.6s cubic-bezier(0.16, 1, 0.3, 1);
    transform-origin: left center;
    flex-shrink: 0;
    min-width: 0;
    max-width: 100%;
  }

  @media (max-width: 1000px) {
    .nav-panels {
      flex-shrink: 0;
      width: auto !important; /* Adapt to Rail or Rail+List */
    }
  }

  .nav-panels.full-width {
    flex: 1;
  }

  /* Mobile Mode Adjustments */
  .mobile-mode .nav-panels {
    width: 100%;
  }

  .mobile-mode .sidebar-wrapper.mobile-drawer {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0,0,0,0.4); /* Subtle Backdrop */
    backdrop-filter: blur(4px);
    z-index: 2000;
    opacity: 0;
    pointer-events: none;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    display: flex !important;
    align-items: center;
    justify-content: center;
    padding: 24px;
    box-sizing: border-box;
  }

  .mobile-mode .sidebar-wrapper.mobile-drawer.open {
    opacity: 1;
    pointer-events: auto;
  }

  .mobile-mode .sidebar-wrapper.mobile-drawer :global(.sidebar) {
    width: 100% !important;
    max-width: 340px;
    height: 70vh !important;
    max-height: 500px;
    transform: scale(0.9) translateY(20px);
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 20px 40px rgba(0,0,0,0.4);
    border-radius: 28px !important;
    border: 1px solid var(--md-outline);
    overflow: hidden;
    background: var(--md-surface);
  }

  .mobile-mode .sidebar-wrapper.mobile-drawer.open :global(.sidebar) {
    transform: scale(1) translateY(0);
  }

  /* Ensure content has padding for bottom nav */
  .mobile-mode :global(.feed-list),
  .mobile-mode .error-view,
  .mobile-mode .loading-overlay {
    padding-bottom: 72px;
  }

  .nav-panels.collapsed {
    max-width: 0 !important;
    opacity: 0;
    pointer-events: none;
    transform: scaleX(0.95);
  }
  .app-container > :global(.sidebar),
  .app-container > :global(.feed-list) {
     flex-shrink: 0;
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
    border-radius: var(--radius-full);
    animation: spin 1s linear infinite;
  }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
