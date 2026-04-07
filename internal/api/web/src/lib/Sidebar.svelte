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
    RefreshCw
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

  $: if (!isMobile) activePopover = null;

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
  <header class="sidebar-header header-bar">
    <div class="logo">
      <span class="full-logo">Gorrson</span>
      <span class="mini-logo">G</span>
    </div>
  </header>
  
  <div class="sidebar-content">
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

    <!-- CATEGORIES SECTION -->
    <div class="section-wrapper">
      <div 
        class="section-header" 
        on:click={(e) => isMobile ? togglePopover('categories', e) : (catsExpanded = !catsExpanded)} 
        role="button" 
        tabindex="0" 
        on:keydown={(e) => e.key === 'Enter' && (catsExpanded = !catsExpanded)}
      >
        <div class="header-left">
          {#if !isMobile}
            {#if catsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
          {/if}
          <span class="mobile-only-icon"><Hash size={18} /></span>
          <span class="section-title-text">{$t('categories')}</span>
        </div>
      </div>
      
      {#if catsExpanded && !isMobile}
        <nav class="categories-group" transition:fly={{ y: -10, duration: 200 }}>
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

      {#if isMobile && activePopover === 'categories'}
        <!-- Popover moved out for overflow reasons -->
      {/if}
    </div>
    
    <div class="divider"></div>

    <!-- SOURCES SECTION -->
    <div class="section-wrapper">
      <div class="section-header">
        <div 
          class="header-left" 
          on:click={(e) => isMobile ? togglePopover('sources', e) : (feedsExpanded = !feedsExpanded)} 
          role="button" 
          tabindex="0" 
          on:keydown={(e) => e.key === 'Enter' && (feedsExpanded = !feedsExpanded)}
        >
          {#if !isMobile}
            {#if feedsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
          {/if}
          <span class="mobile-only-icon"><Globe size={18} /></span>
          <span class="section-title-text">{$t('sources') || 'Sources'}</span>
        </div>
        <div class="header-actions">
          <button class="refresh-btn-inline" on:click|stopPropagation={handleAddClick} title={$t('add_feed')}>
            <Plus size={14} strokeWidth={2.5} />
          </button>
          {#if !isMobile}
            <button class="refresh-btn-inline" on:click|stopPropagation={onRefresh} title="Sync All">
              <RefreshCw size={14} />
            </button>
          {/if}
        </div>
      </div>

      {#if feedsExpanded && !isMobile}
        <nav class="categories-group" transition:fly={{ y: -10, duration: 200 }}>
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
              {/if}
            </button>
          {/each}
        </nav>
      {/if}

      {#if isMobile && activePopover === 'sources'}
        <!-- Popover moved out for overflow reasons -->
      {/if}
    </div>
  </div>

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
            <div class="status">{$t('membership_gold')}</div>
          </div>
        </button>
      </div>
    </div>
  </div>

  <!-- FLOATING POPOVERS (OUTSIDE SCROLL AREA) -->
  {#if isMobile && activePopover === 'categories'}
    <div 
      class="mobile-popover" 
      style="top: {popoverY}px; max-height: {popoverMaxHeight}px;" 
      transition:fly={{ x: -20, duration: 250 }}
    >
      <header class="popover-header">{$t('categories')}</header>
      <div class="popover-content">
        {#each categories as cat}
          <button 
            class="cat-item {activeView === 'cat:'+cat.id ? 'active' : ''}"
            on:click={() => handleNavClick('cat:'+cat.id)}
          >
            <div class="item-left">
              <Hash size={14} />
              <span>{cat.name}</span>
            </div>
            <span class="count">{cat.count}</span>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  {#if isMobile && activePopover === 'sources'}
    <div 
      class="mobile-popover" 
      style="top: {popoverY}px; max-height: {popoverMaxHeight}px;" 
      transition:fly={{ x: -20, duration: 250 }}
    >
      <header class="popover-header">
        <span>{$t('sources') || 'Sources'}</span>
        <div class="header-actions-group">
          <button class="refresh-btn-inline" on:click|stopPropagation={handleAddClick} title={$t('add_feed')}>
            <Plus size={14} strokeWidth={2.5} />
          </button>
          <button class="refresh-btn-inline" on:click|stopPropagation={onRefresh}>
            <RefreshCw size={12} />
          </button>
        </div>
      </header>
      <div class="popover-content">
        {#each feeds as feed}
          <button 
            class="cat-item {activeView === 'src:'+feed.id ? 'active' : ''}"
            on:click={() => handleNavClick('src:'+feed.id)}
          >
            <div class="item-left">
              <Globe size={14} />
              <span class="truncate">{feed.title}</span>
            </div>
            {#if feed.unread_count > 0}
              <span class="count">{feed.unread_count}</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  {/if}
</aside>

<style>
  .sidebar {
    width: var(--sidebar-width);
    background: var(--md-surface);
    height: 100vh;
    display: flex;
    flex-direction: column;
    gap: 0; /* Gap managed by children */
    transition: all var(--layout-transition);
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
    /* No borders here, the path curves away */
  }

  .logo {
    padding: 0 24px;
    font: var(--m3-headline-large);
    font-size: 36px;
    height: 100%;
    line-height: 1;
    font-weight: 900;
    background: linear-gradient(135deg, var(--md-primary), #3B82F6);
    background-clip: text;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    letter-spacing: -2px;
    display: flex;
    align-items: center;
  }

  .mini-logo { display: none; }

  .bottom-actions {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: linear-gradient(to top, var(--md-surface), transparent);
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
    height: 56px;
    padding: 0 24px 0 16px;
    border-radius: var(--radius-full);
    color: var(--md-on-surface-variant);
    font: var(--m3-label-large);
    font-weight: 500;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: background 0.2s, color 0.2s;
    background: transparent;
    border: none;
    width: 100%;
    outline: none;
  }

  .nav-item:hover { 
    background: var(--md-surface-variant); 
    color: var(--md-on-surface); 
  }
  .nav-item.active { 
    background: var(--md-primary-container); 
    color: var(--md-on-primary-container);
    font-weight: 600;
  }

  .item-left { display: flex; align-items: center; gap: 12px; }

  .badge {
    color: var(--md-primary);
    background: var(--md-primary-container);
    padding: 2px 10px;
    border-radius: var(--radius-full);
    font: var(--m3-label-small);
    font-weight: 800;
  }

  .nav-item.active .badge {
    background: var(--md-primary);
    color: var(--md-on-primary);
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
    font: var(--m3-label-medium);
    font-weight: 600;
    margin-left: 12px;
    opacity: 0.9;
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
    height: 40px;
    padding: 0 16px 0 28px;
    font: var(--m3-label-large);
    font-weight: 500;
    color: var(--md-on-surface-variant);
    background: transparent;
    border: none;
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
    border-radius: var(--radius-full);
    transition: all 0.2s;
  }
  .cat-item:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .cat-item.active { 
    background: var(--md-secondary-container); 
    color: var(--md-on-secondary-container); 
    font-weight: 700; 
  }
  
  .count { 
    background: var(--md-surface-variant);
    color: var(--md-on-surface-variant);
    padding: 2px 8px;
    border-radius: var(--radius-full);
    font: var(--m3-label-small);
    font-weight: 600;
  }
  .cat-item.active .count { background: var(--md-primary); color: var(--md-on-primary); }

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
    padding: 8px 12px;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: all 0.2s;
  }

  .profile-main {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px;
    border-radius: var(--radius-full);
    cursor: pointer;
    min-width: 0;
    transition: all 0.2s;
    border: none;
    background: transparent;
    text-align: left;
  }
  .profile-main:hover {
    background: var(--md-secondary-container);
  }

  .avatar {
    width: 32px;
    height: 32px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    font: var(--m3-label-large);
    font-weight: 800;
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

  .user-info { flex: 1; min-width: 0; }
  .username { font: var(--m3-label-large); font-weight: 700; color: var(--md-primary); text-overflow: ellipsis; white-space: nowrap; overflow: hidden; }
  .status { font: var(--m3-label-small); color: var(--md-on-surface-variant); opacity: 0.8; }



  .mobile-only-icon {
    display: none;
  }

  .section-wrapper {
    position: relative;
  }

  .mobile-popover {
    position: fixed; /* Fixed to avoid scroll-parent issues */
    left: 72px;      /* Tighter to the rail */
    width: 260px;    /* Slightly wider for better text flow */
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-xl);
    box-shadow: 12px 4px 40px rgba(0,0,0,0.25);
    z-index: 9999;
    display: flex;
    flex-direction: column;
    animation: popIn 0.2s cubic-bezier(0, 0, 0.2, 1);
  }

  @keyframes popIn {
    from { opacity: 0; transform: translateX(-10px); }
    to { opacity: 1; transform: translateX(0); }
  }

  .popover-header {
    padding: 16px 20px;
    font: var(--m3-label-large);
    font-weight: 800;
    color: var(--md-primary);
    border-bottom: 1px solid var(--md-outline);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .popover-content {
    padding: 8px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .popover-content .cat-item {
    padding: 0 12px;
    opacity: 1;
    height: 44px;
  }

  .mobile-popover .cat-item span,
  .mobile-popover .count,
  .mobile-popover .truncate {
    display: block !important;
  }

  .mobile-popover .item-left {
    gap: 12px;
    justify-content: flex-start !important;
  }

  .mobile-popover .cat-item {
    justify-content: space-between !important;
    padding: 0 16px !important;
    border-radius: var(--radius-m);
  }

  .header-actions-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  /* Compact Mode (Tablet & Mobile) */
  @media (max-width: 1000px) {
    .sidebar {
      padding: 16px 0;
      width: 72px !important;
      z-index: 1000;
      border-right: 1px solid var(--md-outline);
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
  }
</style>

