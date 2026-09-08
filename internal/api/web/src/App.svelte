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
  import RecipeBuilder from './lib/RecipeBuilder.svelte';
  import MobileNav from './lib/MobileNav.svelte';
  import { success as toastSuccess, error as toastError } from './lib/toast.js';
  import { formatTimeAgo } from './lib/utils.js';
  import { Inbox, Star, Bookmark, Archive, Shield, BarChart2 } from 'lucide-svelte';
  import { t } from './lib/i18n.js';
  import { onMount } from 'svelte';
  import { fetchNews, markAsRead, markAllRead, fetchFeeds, toggleFavorite } from './lib/api.js';
  import { generateTonal } from './lib/theme.js';

  /** @type {string | number | null} */
  let selectedId = null;
  let searchQuery = "";
  let isMobile = typeof window !== 'undefined' ? window.innerWidth <= 768 : false;
  let isSidebarMini = typeof window !== 'undefined' ? window.innerWidth <= 1000 : false;
  let historyPushed = false;
  
  let offset = 0;
  const LIMIT = 50;
  let allLoaded = false;
  let loadingMore = false;
  let scrollRead = localStorage.getItem('gorrson-scroll-read') === 'true';
  let unreadOnly = localStorage.getItem('gorrson-unread-only') === 'true';
  let sortMode = "newest"; // "newest", "oldest"
  let cardStyle = localStorage.getItem('gorrson-card-style') || 'default';
  let readerMode = localStorage.getItem('gorrson-reader-mode') || 'side-by-side';

  $: localStorage.setItem('gorrson-scroll-read', scrollRead.toString());
  $: localStorage.setItem('gorrson-unread-only', unreadOnly.toString());
  $: localStorage.setItem('gorrson-card-style', cardStyle);
  $: localStorage.setItem('gorrson-reader-mode', readerMode);

  const STOP_WORDS = new Set(['de', 'a', 'el', 'la', 'en', 'y', 'que', 'os', 'un', 'una', 'con', 'para', 'por', 'lo', 'las', 'los', 'su', 'al', 'se', 'del', 'the', 'and', 'with', 'that', 'this', 'from']);

  const updateMobile = () => {
    isMobile = window.innerWidth <= 768;
    isSidebarMini = window.innerWidth <= 1000;
  };
  /** @type {Array<any>} */
  let articles = [];
  /** @type {Array<any>} */
  let feeds = [];
  let feedsLoaded = false;
  let loading = true;
  /** @type {string | null} */
  let error = null;
  let isAuthenticated = false;
  let currentView = "inbox";
  /** @type {string | null} */
  let activeCategory = null;
  /** @type {string | null} */
  let activeSource = null;
  let showAddFeed = false;
  /** @type {any} */
  let recipeBuilderFeed = null;
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
      if (isAuthenticated && !loading && !loadingMore && !searchQuery && document.visibilityState === 'visible') {
        loadData();
      }
    }, 5 * 60 * 1000);

    // Detectar cuando el usuario pulsa "Atrás" en el ratón o navegador
    /** @param {PopStateEvent} e */
    const handlePopState = (e) => {
      if (selectedId) {
        selectedId = null;
        historyPushed = false;
      }
    };
    window.addEventListener('popstate', handlePopState);

    return () => {
      clearInterval(interval);
      clearInterval(pollInterval);
      window.removeEventListener('popstate', handlePopState);
    };
  });

  $: lastSyncFormatted = formatTimeAgo(lastSyncDate);

  $: totalUnread = (feeds || []).reduce((acc, f) => acc + (f.unread_count || 0), 0);
  
  $: navItems = [
    { id: "inbox", name: "Inbox", badge: totalUnread, icon: Inbox },
    { id: "favorites", name: "Favorites", icon: Star },
    { id: "insights", name: "Insights", icon: BarChart2 },
  ];

  $: viewTitle = activeCategory ? activeCategory : (activeSource ? (feeds.find(f => f.id === activeSource)?.title || "Feed") : $t(currentView));

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
        feedsLoaded ? Promise.resolve(feeds) : fetchFeeds()
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
      if (!feedsLoaded) {
        feeds = feedsData || [];
        feedsLoaded = true;
      }
      
      // Auto-selection only for full desktop
      if (articles.length > 0 && !selectedId && !append && !isMobile && !isSidebarMini && readerMode === 'side-by-side') {
        selectedId = articles[0].id;
      }
      lastSyncDate = new Date();
    } catch (e) {
      const err = /** @type {Error} */ (e);
      error = err.message;
      if (err.message === 'Unauthorized') isAuthenticated = false;
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  let notifiedIds = new Set();
  /** @param {Array<any>} newEntries */
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

  /** @param {string} viewId */
  function handleViewChange(viewId) {
    selectedId = null;
    historyPushed = false;
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

  let prevUnreadOnly;
  let prevSortMode;
  $: if (isAuthenticated && (unreadOnly !== prevUnreadOnly || sortMode !== prevSortMode)) {
    prevUnreadOnly = unreadOnly;
    prevSortMode = sortMode;
    loadData();
  }

  // Pre-filter search logic (client side)
  $: filteredArticles = (() => {
    let list = articles;
    if (unreadOnly) {
      list = list.filter(a => !a.is_read);
    }

    if (!searchQuery || searchQuery.trim().length === 0) return list;

    const tokens = searchQuery.toLowerCase()
      .split(/\s+/)
      .filter(word => word.length > 2 && !STOP_WORDS.has(word));

    if (tokens.length === 0 && searchQuery.trim().length > 0) {
      // If we have query but all are stop words, just do a literal match of the whole query
      return list.filter(a => a.title.toLowerCase().includes(searchQuery.toLowerCase()));
    }

    if (tokens.length === 0) return list;

    return list.filter(article => {
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

  /** @param {string|number} id */
  async function handleMarkAsRead(id) {
    if (articles.find(a => a.id === id)?.is_read) return;
    
    // Update local state immediately for instant feedback
    articles = articles.map(a => a.id === id ? {...a, is_read: true} : a);
    
    try {
      await markAsRead(id);
      // If we're in unread only mode and the current list becomes too short, fetch more
      if (unreadOnly && filteredArticles.length < 5 && !allLoaded) {
        loadData(true);
      }
    } catch (e) {
      // Revert if API fails
      articles = articles.map(a => a.id === id ? {...a, is_read: false} : a);
      toastError('Failed to mark as read');
    }
  }

  /** @param {string|number} id */
  async function handleSelect(id) {
    // Guardar estado en el historial solo si es el primer artículo que abrimos
    if (!selectedId) {
      window.history.pushState({ articleOpen: true }, "");
      historyPushed = true;
    }
    selectedId = id;
    handleMarkAsRead(id);
  }

  async function handleMarkAllRead(isGlobal = false) {
    const unreadCount = articles.filter(a => !a.is_read).length;
    if (!isGlobal && unreadCount === 0) return;

    // 1. Update local state immediately
    const previousArticles = [...articles];
    articles = articles.map(a => ({...a, is_read: true}));

    try {
      let params = {};
      if (!isGlobal) {
        params = {
          category: activeCategory,
          feed_id: activeSource,
          search: searchQuery,
          limit: previousArticles.length 
        };
      }
      
      await markAllRead(params);
      toastSuccess(isGlobal ? 'All system news marked as read' : `Marked ${unreadCount} items as read`);

      // 2. Reload data after a small delay
      setTimeout(() => {
        loadData();
      }, 300);
    } catch (e) {
      articles = previousArticles;
      toastError('Failed to mark as read');
    }
  }
  /** @param {string|number} id */
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

  /** @param {any} article */
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

  /** @param {string} url */
  async function copyToClipboard(url) {
    await navigator.clipboard.writeText(url);
    toastSuccess($t('link_copied'));
  }

  function handleLoginSuccess() {
    isAuthenticated = true;
    loadData();
  }

  async function handleLogout() {
    try {
      const token = localStorage.getItem('token');
      await fetch('/logout', {
        method: 'POST',
        headers: token ? { 'Authorization': `Bearer ${token}` } : {},
      });
    } catch (_) {}
    localStorage.removeItem('token');
    localStorage.removeItem('refresh_token');
    isAuthenticated = false;
  }

  function handleAddFeed() {
    showAddFeed = true;
  }

  /** @param {string} feedTitle */
  function handleAddFeedSave(feedTitle) {
    showAddFeed = false;
    feedsLoaded = false;
    toastSuccess(`Feed "${feedTitle || 'nuevo feed'}" añadado correctamente ✓`);
    loadData();
  }

  function handleAddFeedCancel() {
    showAddFeed = false;
  }

  let fullscreenReader = false;
  const toggleFullscreen = () => fullscreenReader = !fullscreenReader;

  // Auto-collapse feed list if user shrinks the browser
  $: if (isMobile && readerMode === 'full-list') {
    readerMode = 'side-by-side';
  }
</script>

{#if !isAuthenticated}
  <Login onLoginSuccess={handleLoginSuccess} />
{:else}
  <main class="app-container {selectedId ? 'has-selection' : ''} {isMobile ? 'mobile-mode' : ''}">
    {#if !isMobile || isSourcesDrawerOpen}
      <div 
        class="sidebar-wrapper {fullscreenReader ? 'hidden-desktop' : ''}" 
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

    <div class="main-content-wrapper">
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
          <FeedList 
            bind:searchQuery
            bind:cardStyle
            title={viewTitle}
            articles={filteredArticles} 
            selectedId={selectedId} 
            hasSelection={!!selectedId}
            bind:readerMode
            onSelect={handleSelect} 
            onMarkAsRead={handleMarkAsRead}
            onToggleFavorite={handleToggleFavorite}
            onShare={handleShare}
            onMarkAllRead={handleMarkAllRead}
            onLoadMore={handleLoadMore}
            {loadingMore}
            {allLoaded}
            {scrollRead}
            bind:unreadOnly
            bind:sortMode
          >
            <svelte:fragment slot="reader">
              {#if selectedId}
                <PostViewer 
                  article={selectedArticle} 
                  {articles}
                  onSelect={handleSelect}
                  onBack={() => {
                    if (historyPushed) {
                      window.history.back(); // Esto dispara el popstate de arriba
                      historyPushed = false;
                    } else {
                      selectedId = null;
                    }
                  }}
                  isFullscreen={fullscreenReader}
                  onToggleFullscreen={toggleFullscreen}
                  onToggleFavorite={handleToggleFavorite}
                />
              {/if}
            </svelte:fragment>
          </FeedList>
        {:else if currentView === 'filters'}
          <Filters />
        {:else if currentView === 'insights'}
          <Insights />
        {:else if currentView === 'settings'}
          <Settings 
            {feeds} 
            username={localStorage.getItem('username')} 
            onLogout={handleLogout} 
            onRefresh={loadData}
            onFeedsChanged={() => { feedsLoaded = false; }}
            bind:scrollRead
            bind:unreadOnly
            bind:cardStyle
            {isMobile}
            onGenerateRecipe={(feed) => {
              const latest = articles.find(a => a.feed_id === feed.id);
              recipeBuilderFeed = { 
                ...feed, 
                sample_url: latest ? latest.url : feed.site_url 
              };
            }}
          />
        {/if}
      {/if}
    </div>

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

    {#if recipeBuilderFeed}
      <RecipeBuilder 
        show={!!recipeBuilderFeed} 
        feed={recipeBuilderFeed} 
      onClose={(saved) => { 
        recipeBuilderFeed = null; 
        if (saved === true) loadData(); 
      }} 
      />
    {/if}

    <Toaster />
  </main>
{/if}

<style>
  .app-container {
    display: flex;
    height: 100vh;
    overflow: hidden;
    background: var(--md-background);
  }

  .app-container.mobile-mode {
    display: block;
    height: auto;
    overflow: visible;
  }

  .sidebar-wrapper {
    flex-shrink: 0;
    transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
    width: var(--sidebar-width);
    z-index: 100;
  }

  .sidebar-wrapper.hidden-desktop {
    width: 0;
    opacity: 0;
    pointer-events: none;
    transform: translateX(-20px);
  }

  .main-content-wrapper {
    flex: 1;
    display: flex;
    min-width: 0;
    height: 100%;
    position: relative;
    background: var(--md-background);
  }

  .mobile-mode .main-content-wrapper {
    display: block;
    height: auto;
  }

  .mobile-mode .sidebar-wrapper.mobile-drawer {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
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
