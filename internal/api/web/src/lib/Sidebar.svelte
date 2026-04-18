<script>
  import { 
    Plus, 
    Inbox, 
    Star, 
    Bookmark, 
    Archive, 
    Settings, 
    Hash,
    LogOut,
    ChevronDown,
    ChevronRight,
    Globe,
    RefreshCw,
    X
  } from 'lucide-svelte';
  import { locale, t } from './i18n.js';
  import { fly } from 'svelte/transition';

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };

  /** @type {Array<{id: string, name: string, badge?: number, active?: boolean, icon: any}>} */
  export let navItems = [];
  
  /** @type {Array<{id: string, name: string, count: number}>} */
  export let categories = [];

  /** @type {Array<any>} */
  export let feeds = [];

  /** @type {string} */
  export let username = localStorage.getItem('username') || "User";

  export let onViewChange = (viewId) => {};
  export let onAddFeed = () => {};
  export let onRefresh = () => {};
  export let activeView = 'inbox';
  export let isMobile = false;
  export let isMini = false;
  export let lastSync = "";

  let catsExpanded = true;
  let feedsExpanded = true;
  let activePopover = null; // 'categories' or 'sources' or null
  let popoverY = 0;
  let popoverMaxHeight = 400;

  const togglePopover = (type, event) => {
    if (activePopover === type) activePopover = null;
    else {
      activePopover = type;
      if (event) {
        const rect = event.currentTarget.getBoundingClientRect();
        const vh = window.innerHeight;
        
        let y = rect.top;
        let spaceBelow = vh - y - 24; // 24px safety margin at bottom

        // If very little space below (< 300px), we shift up to provide at least 400px of view
        if (spaceBelow < 300) {
           const idealHeight = Math.min(vh - 40, 480);
           y = vh - idealHeight - 20;
           spaceBelow = vh - y - 24;
        }

        popoverY = Math.max(10, y);
        popoverMaxHeight = spaceBelow;
      }
    }
  };

  $: if (!isMini && !isMobile) activePopover = null;

  const handleNavClick = (id) => {
    activePopover = null;
    showUserMenu = false;
    onViewChange(id);
  };
  
  const handleAddClick = () => {
    onAddFeed();
  };

  let showUserMenu = false;
  const toggleUserMenu = () => {
    showUserMenu = !showUserMenu;
  };

  const handleLogout = () => {
    showUserMenu = false;
    localStorage.removeItem('token');
    window.location.reload();
  };
</script>

<aside class="sidebar">
  {#if !isMobile}
      <header class="sidebar-header">
        <div class="logo-container">
          <h1 class="full-logo">Gorrson</h1>
          <p class="logo-subtitle">Reader & Archive</p>
          <span class="mini-logo">G</span>
        </div>
    </header>
  {/if}
  
  <div class="sidebar-content">
    {#if !isMobile}
      <nav class="nav-group">
        {#each navItems as item}
          <button 
            class="nav-item {activeView === item.id ? 'active' : ''}"
            on:click={() => handleNavClick(item.id)}
          >
            <div class="item-left">
              <svelte:component this={item.icon} size={18} strokeWidth={2} />
              <span>{$t(item.id)}</span>
            </div>
            {#if item.badge}
              <span class="badge">{item.badge}</span>
            {/if}
          </button>
        {/each}
      </nav>

      <div class="divider"></div>
    {/if}

    <!-- CATEGORIES SECTION -->
    <div class="section-wrapper">
      {#if isMobile}
        <button class="mobile-toggle-btn" on:click={() => catsExpanded = !catsExpanded}>
          <span class="toggle-title">{$t('categories')}</span>
          <div class="toggle-icon">
            {#if catsExpanded}<ChevronDown size={18} />{:else}<ChevronRight size={18} />{/if}
          </div>
        </button>
      {:else}
        <div 
          class="section-header" 
          on:click={(e) => isMini ? togglePopover('categories', e) : (catsExpanded = !catsExpanded)} 
          role="button" 
          tabindex="0" 
          on:keydown={(e) => e.key === 'Enter' && (catsExpanded = !catsExpanded)}
        >
          <div class="header-left">
            {#if !isMini}
              {#if catsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
            {/if}
            <span class="mobile-only-icon"><Hash size={18} /></span>
            <span class="section-title-text">{$t('categories')}</span>
          </div>
        </div>
      {/if}
      
      {#if (catsExpanded && !isMini && !isMobile) || (isMobile && catsExpanded)}
        <nav class="categories-group {isMobile ? 'mobile-expanded' : ''}" transition:fly={{ y: -10, duration: 200 }}>
          {#each categories as cat}
            <button 
              class="cat-item {activeView === 'cat:'+cat.id ? 'active' : ''}"
              on:click={() => handleNavClick('cat:'+cat.id)}
            >
              <div class="item-left">
                <Hash size={14} opacity={0.5} />
                <span>{cat.name}</span>
              </div>
              <span class="count">{cat.count}</span>
            </button>
          {/each}
        </nav>
      {/if}
    </div>
    
    {#if !isMobile}
      <div class="divider"></div>
    {/if}

    <!-- SOURCES SECTION -->
    <div class="section-wrapper">
      {#if isMobile}
        <button class="mobile-toggle-btn" on:click={() => feedsExpanded = !feedsExpanded}>
          <span class="toggle-title">{$t('sources') || 'Sources'}</span>
          <div class="toggle-icon">
            {#if feedsExpanded}<ChevronDown size={18} />{:else}<ChevronRight size={18} />{/if}
          </div>
        </button>
      {:else}
        <div class="section-header">
          <div 
            class="header-left" 
            on:click={(e) => isMini ? togglePopover('sources', e) : (feedsExpanded = !feedsExpanded)} 
            role="button" 
            tabindex="0" 
            on:keydown={(e) => e.key === 'Enter' && (feedsExpanded = !feedsExpanded)}
          >
            {#if !isMini}
              {#if feedsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
            {/if}
            <span class="mobile-only-icon"><Globe size={18} /></span>
            <span class="section-title-text">{$t('sources') || 'Sources'}</span>
          </div>
          <div class="header-actions">
            <button class="refresh-btn-inline" on:click|stopPropagation={handleAddClick} title={$t('add_feed')}>
              <Plus size={14} strokeWidth={2.5} />
            </button>
            {#if !isMini}
              <button class="refresh-btn-inline" on:click|stopPropagation={onRefresh} title="Sync All">
                <RefreshCw size={14} />
              </button>
            {/if}
          </div>
        </div>
      {/if}

      {#if (feedsExpanded && !isMini && !isMobile) || (isMobile && feedsExpanded)}
        <nav class="categories-group {isMobile ? 'mobile-expanded' : ''}" transition:fly={{ y: -10, duration: 200 }}>
          {#each feeds as feed}
            <button 
              class="cat-item {activeView === 'src:'+feed.id ? 'active' : ''}"
              on:click={() => handleNavClick('src:'+feed.id)}
            >
              <div class="item-left">
                <Globe size={14} opacity={0.5} />
                <span class="truncate">{feed.title}</span>
              </div>
              {#if feed.unread_count > 0}
                <span class="count">{feed.unread_count}</span>
              {:else}
                <span class="count">0</span>
              {/if}
            </button>
          {/each}
        </nav>
      {/if}
    </div>
  </div>

  {#if !isMobile}
    <div class="bottom-actions">
      <div class="user-profile-container">
        {#if showUserMenu}
          <div class="user-popup" transition:fly={{ y: 20, duration: 200 }}>
            <button class="popup-item {activeView === 'settings' ? 'active' : ''}" on:click={() => handleNavClick('settings')}>
              <Settings size={16} />
              <span>{$t('settings') || 'Settings'}</span>
            </button>
            <button class="popup-item" on:click={() => { toggleLanguage(); showUserMenu = false; }}>
              <Globe size={16} />
              <span>{$t('language') || 'Language'}: { $locale.toUpperCase() }</span>
            </button>
            <div class="divider"></div>
            <button class="popup-item logout-item" on:click={handleLogout}>
              <LogOut size={16} />
              <span>{$t('logout') || 'Log Out'}</span>
            </button>
          </div>
        {/if}

        <div class="user-profile {showUserMenu ? 'menu-open' : ''}">
          <button class="profile-main" on:click={toggleUserMenu} title="User menu">
            <div class="avatar">{username[0]?.toUpperCase() || 'U'}</div>
            <div class="user-info">
              <div class="username">{username}</div>
              <div class="status">{$t('synced')} {lastSync}</div>
            </div>
          </button>
        </div>
      </div>
    </div>
  {/if}

</aside>

{#if isMini && activePopover}
  <div 
    class="sidebar-popover" 
    style="top: {popoverY}px; max-height: {popoverMaxHeight}px;"
    transition:fly={{ x: -20, duration: 200 }}
  >
    <div class="popover-header">
      <span class="popover-title">{$t(activePopover)}</span>
      <button class="close-popover" on:click={() => activePopover = null}>
        <X size={16} />
      </button>
    </div>
    
    <div class="popover-scroll scroll-area">
      {#if activePopover === 'categories'}
        {#each categories as cat}
          <button 
            class="cat-item {activeView === 'cat:'+cat.id ? 'active' : ''}"
            on:click={() => handleNavClick('cat:'+cat.id)}
          >
            <div class="item-left">
              <Hash size={14} opacity={0.5} />
              <span>{cat.name}</span>
            </div>
            <span class="count">{cat.count}</span>
          </button>
        {/each}
      {:else if activePopover === 'sources'}
        {#each feeds as feed}
          <button 
            class="cat-item {activeView === 'src:'+feed.id ? 'active' : ''}"
            on:click={() => handleNavClick('src:'+feed.id)}
          >
            <div class="item-left">
              <Globe size={14} opacity={0.5} />
              <span class="truncate">{feed.title}</span>
            </div>
            <span class="count">{feed.unread_count || 0}</span>
          </button>
        {/each}
      {/if}
    </div>
  </div>
  
  <button type="button" class="popover-backdrop" on:click={() => activePopover = null} aria-label="Close"></button>
{/if}

<style>
  .sidebar {
    width: var(--sidebar-width);
    background: var(--md-surface-variant);
    height: 100vh;
    display: flex;
    flex-direction: column;
    gap: 0; /* Gap managed by children */
    transition: all var(--layout-transition);
    overflow: hidden;
    border-top-right-radius: 24px;
    border-bottom-right-radius: 24px;
  }

  .sidebar-header {
    height: 64px !important;
    min-height: 64px !important;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    background: var(--md-surface);
    padding: 0;
    box-sizing: border-box;
    background: transparent;
  }

  .logo-container {
    padding: 24px 24px 16px 24px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 100%;
  }
  .full-logo {
    font-family: var(--font-display);
    font-size: 22px;
    font-weight: 900;
    color: var(--md-on-surface);
    margin: 0;
    line-height: 1.1;
    letter-spacing: -0.5px;
  }
  .logo-subtitle {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 2px;
    font-weight: 800;
    color: var(--md-on-surface-variant);
    margin: 0;
  }

  .mini-logo { display: none; }

  .bottom-actions {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 0;
    background: linear-gradient(to top, var(--md-surface-variant), transparent);
  }

  .header-actions {
    display: flex;
    gap: 4px;
    margin-left: auto;
  }

  .nav-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px; /* Restore padding for nav items */
  }

  .nav-item {
    height: auto;
    padding: 10px 16px;
    border-radius: var(--radius-full);
    color: var(--md-on-surface-variant);
    font-size: 11px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
    background: transparent;
    border: none;
    width: 100%;
    outline: none;
    margin-bottom: 4px;
  }

  .nav-item:hover { 
    background: color-mix(in srgb, var(--md-on-surface) 5%, transparent); 
    color: var(--md-on-surface); 
  }
  .nav-item.active { 
    background: var(--md-primary); 
    color: var(--md-on-primary);
    box-shadow: 0 4px 12px var(--shadow-color);
  }

  .item-left { display: flex; align-items: center; gap: 12px; }

  .badge {
    background: color-mix(in srgb, var(--md-on-surface) 10%, transparent);
    color: inherit;
    padding: 2px 8px;
    border-radius: var(--radius-full);
    font-size: 10px;
    font-weight: 800;
  }

  .nav-item.active .badge {
    background: rgba(255,255,255,0.2);
  }

  .divider { height: 1px; background: var(--md-outline); opacity: 0.15; margin: 12px 16px; }

  .section-header {
    width: 100%;
    background: transparent;
    border: none;
    display: flex;
    align-items: center;
    padding: 14px 16px 14px 28px;
    cursor: pointer;
    color: var(--md-on-surface-variant);
    transition: all 0.2s;
  }

  .section-title-text {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    font-weight: 800;
    color: var(--md-on-surface-variant);
    opacity: 0.7;
    margin-left: 12px;
  }

  .refresh-btn-inline {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 4px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    margin-left: auto;
    opacity: 0.5;
  }
  .refresh-btn-inline:hover {
    background: var(--md-secondary-container);
    color: var(--md-primary);
    opacity: 1;
    transform: rotate(180deg);
  }

  .truncate {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 160px;
  }

  .categories-group { 
    display: flex; 
    flex-direction: column; 
    gap: 2px; 
    padding: 0 12px; 
  }

  .cat-item {
    height: auto;
    padding: 10px 16px 10px 24px;
    border-radius: var(--radius-full);
    color: var(--md-on-surface-variant);
    font-size: 11px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
    background: transparent;
    border: none;
    width: 100%;
    outline: none;
    margin-bottom: 2px;
  }
  .cat-item:hover { background: color-mix(in srgb, var(--md-on-surface) 5%, transparent); color: var(--md-on-surface); }
  .cat-item.active { 
    background: var(--md-primary); 
    color: var(--md-on-primary); 
    box-shadow: 0 4px 12px var(--shadow-color);
  }
  
  .count { 
    background: color-mix(in srgb, var(--md-on-surface) 10%, transparent);
    color: inherit;
    padding: 2px 6px;
    border-radius: var(--radius-full);
    font-size: 10px;
    font-weight: 800;
  }
  .cat-item.active .count { background: rgba(255,255,255,0.2); }

  .user-profile-container {
    position: relative;
    width: 100%;
  }

  .user-popup {
    position: absolute;
    bottom: calc(100% + 12px);
    left: 0;
    right: 0;
    background: var(--md-surface, #fdfdfd);
    border: 1px solid var(--md-outline, #ccc);
    border-radius: var(--radius-xl);
    box-shadow: 0 4px 20px rgba(0,0,0,0.15), 0 8px 40px rgba(0,0,0,0.1);
    padding: 8px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow: hidden;
  }

  /* Popover for Mini Sidebar */
  .sidebar-popover {
    position: fixed;
    left: 76px;
    width: 240px;
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    box-shadow: 8px 0 32px rgba(0,0,0,0.3);
    z-index: 3000;
    display: flex;
    flex-direction: column;
    padding: 8px;
    overflow: hidden;
  }

  .popover-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid var(--md-outline);
    margin-bottom: 8px;
  }

  .popover-title {
    font: var(--m3-label-large);
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--md-primary);
  }

  .close-popover {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 4px;
    border-radius: var(--radius-full);
  }
  .close-popover:hover { background: var(--md-surface-variant); }

  .popover-scroll {
    overflow-y: auto;
    padding-right: 4px;
  }

  .popover-backdrop {
    position: fixed;
    inset: 0;
    z-index: 2999;
    background: transparent;
    border: none;
    padding: 0;
    cursor: default;
  }

  .sidebar-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 12px 0;
    /* Border delegated to feed-content for unibody path */
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: none; /* Firefox */
  }
  .sidebar-content::-webkit-scrollbar {
    display: none; /* Chrome/Safari */
  }

  .popup-item {
    width: 100%;
    padding: 10px 16px;
    border: none;
    background: transparent;
    color: var(--md-on-surface);
    font: var(--m3-label-large);
    text-align: left;
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
    border-radius: var(--radius-full);
    transition: all 0.2s;
  }
  .popup-item :global(svg) {
    color: var(--md-primary);
    opacity: 0.9;
  }
  .popup-item:hover {
    background: var(--md-surface-variant);
  }
  .popup-item.active {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    font-weight: 700;
  }
  .popup-item.active :global(svg) {
    color: var(--md-primary);
    opacity: 1;
  }

  .user-profile {
    padding: 16px 24px;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s;
    border-top: 1px solid var(--md-outline);
  }

  .profile-main {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 12px;
    border-radius: var(--radius-full);
    cursor: pointer;
    min-width: 0;
    transition: all 0.2s;
    border: none;
    background: transparent;
    text-align: left;
    padding: 0;
  }

  .avatar {
    width: 40px;
    height: 40px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
    font-weight: 900;
    flex-shrink: 0;
    position: relative;
    transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    overflow: hidden;
  }

  /* Overlay for hover visibility */
  .avatar::after {
    content: "";
    position: absolute;
    inset: 0;
    background: white;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .profile-main:hover .avatar::after {
    opacity: 0.15;
  }

  .user-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px;}
  .username { font-size: 11px; font-weight: 800; color: var(--md-on-surface); text-overflow: ellipsis; white-space: nowrap; overflow: hidden; }
  .status { font-size: 9px; text-transform: uppercase; letter-spacing: 1px; font-weight: 700; color: var(--md-on-surface-variant); opacity: 0.8; }



  .mobile-only-icon {
    display: none;
  }

  /* Compact Mode (Tablet & Mobile) */
  @media (max-width: 1000px) {
    .sidebar {
      padding: 16px 0;
      width: 72px !important;
      z-index: 1000;
      border-right: none !important;
      display: flex !important;
      flex-direction: column !important;
      align-items: center !important;
      box-sizing: border-box !important;
    }
    .sidebar-header,
    .sidebar-content,
    .bottom-actions {
      width: 100% !important;
      display: flex !important;
      flex-direction: column !important;
      align-items: center !important;
      padding-left: 0 !important;
      padding-right: 0 !important;
      box-sizing: border-box !important;
    }
    .full-logo, 
    .logo-subtitle,
    .nav-item span, 
    .cat-item span, 
    .section-title-text,
    .user-info, 
    .badge, 
    .count,
    .divider {
      display: none !important;
    }

    .nav-group {
      gap: 12px;
    }

    /* Unified Rail Item Style */
    .nav-item, 
    .section-header,
    .profile-main {
      width: 48px !important;
      height: 48px !important;
      padding: 0 !important;
      margin: 8px 0 !important; /* Vertical spacing only */
      display: flex !important;
      align-items: center !important;
      justify-content: center !important;
      border-radius: var(--radius-full) !important;
      background: transparent;
      border: none;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      cursor: pointer;
      position: relative;
    }

    .nav-item:hover, 
    .section-header:hover, 
    .profile-main:hover {
      background: var(--md-surface-variant);
    }

    .nav-item.active {
      background: var(--md-primary-container) !important;
      color: var(--md-on-primary-container) !important;
    }

    .header-left,
    .item-left,
    .mobile-only-icon {
      width: 100% !important;
      height: 100% !important;
      display: flex !important;
      align-items: center !important;
      justify-content: center !important;
      margin: 0 !important;
      padding: 0 !important;
      gap: 0 !important; /* Extremely important to prevent off-centering */
    }

    .header-actions {
      display: none !important;
    }

    .divider {
      margin: 12px 10px;
      display: none !important;
    }

    .mini-logo {
      display: block !important;
      font-size: 24px;
      font-weight: 900;
      letter-spacing: -1px;
    }
    .logo {
      padding: 0 !important;
      margin: 0 !important;
      width: 100%;
      height: 100%;
      display: flex !important;
      align-items: center !important;
      justify-content: center !important;
    }
    .bottom-actions {
      align-items: center;
      padding: 12px 0;
      width: 100%;
    }
    .user-profile {
      display: flex;
      flex-direction: column;
      align-items: center;
      width: 100%;
      background: transparent !important;
      border: none !important;
      margin: 0;
      padding: 0;
    }
    .profile-main {
      margin: 8px 0 !important;
    }
    .user-popup {
      left: 64px;
      bottom: 20px;
      width: 180px;
      box-shadow: 4px 4px 20px rgba(0,0,0,0.3);
    }
    .avatar {
      width: 32px;
      height: 32px;
      font-size: 14px;
      border-radius: var(--radius-full);
      box-shadow: 0 0 0 2px var(--md-outline);
    }

    /* Force visibility in popovers even if general rule hides it */
    .sidebar-popover .cat-item span,
    .sidebar-popover .count {
      display: block !important;
    }
    .sidebar-popover .item-left {
      width: auto !important;
      justify-content: flex-start !important;
      gap: 12px !important;
    }
  }


  .mobile-expanded {
    display: flex !important;
    flex-direction: column;
    padding: 0 12px !important;
    gap: 4px !important;
  }

  .mobile-expanded .cat-item {
    height: 48px !important;
    border-radius: 12px !important;
    padding: 0 16px !important;
    width: 100% !important;
  }

  .mobile-expanded .cat-item span {
    display: block !important;
    font-size: 15px !important;
    font-weight: 500;
  }

  .mobile-toggle-btn {
    width: 100%;
    background: transparent;
    border: none;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    cursor: pointer;
    color: var(--md-on-surface);
    transition: background 0.2s;
    border-radius: 16px;
    margin-bottom: 4px;
  }

  .mobile-toggle-btn:active {
    background: var(--md-surface-variant);
  }

  .toggle-title {
    font: var(--m3-title-small);
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--md-primary);
    opacity: 0.8;
  }

  .toggle-icon {
    color: var(--md-primary);
    opacity: 0.7;
    display: flex;
    align-items: center;
  }
</style>
