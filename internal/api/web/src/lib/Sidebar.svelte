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

  let catsExpanded = true;
  let feedsExpanded = true;

  const handleNavClick = (id) => {
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
    localStorage.removeItem('token');
    window.location.reload();
  };
</script>

<aside class="sidebar">
  <header class="sidebar-header header-bar">
    <div class="logo">
      <span class="full-logo">GORRSON</span>
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
    <div class="section-header" on:click={() => catsExpanded = !catsExpanded} role="button" tabindex="0" on:keydown={(e) => e.key === 'Enter' && (catsExpanded = !catsExpanded)}>
      <div class="header-left">
        {#if catsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
        <span class="section-title-text">{$t('categories')}</span>
      </div>
    </div>
    
    {#if catsExpanded}
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

    <div class="divider"></div>

    <!-- SOURCES SECTION -->
    <div class="section-header" on:click={() => feedsExpanded = !feedsExpanded} role="button" tabindex="0" on:keydown={(e) => e.key === 'Enter' && (feedsExpanded = !feedsExpanded)}>
      <div class="header-left">
        {#if feedsExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
        <span class="section-title-text">{$t('sources') || 'Sources'}</span>
      </div>
      <button class="refresh-btn-inline" on:click|stopPropagation={onRefresh} title="Sync All">
        <RefreshCw size={12} />
      </button>
    </div>

    {#if feedsExpanded}
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
  </div>

  <div class="bottom-actions">
    <button class="add-btn" on:click={handleAddClick}>
      <Plus size={20} strokeWidth={3} />
      <span>{$t('add_feed')}</span>
    </button>

    <div class="user-profile-container">
      {#if showUserMenu}
        <div class="user-popup" transition:fly={{ y: 20, duration: 200 }}>
          <button class="popup-item" on:click={handleLogout}>
            <LogOut size={16} />
            <span>{$t('logout') || 'Log Out'}</span>
          </button>
          <button class="popup-item" on:click={() => { toggleLanguage(); showUserMenu = false; }}>
            <span>{$t('language') || 'Language'}: { $locale.toUpperCase() }</span>
          </button>
        </div>
      {/if}

      <div class="user-profile {showUserMenu ? 'menu-open' : ''}">
        <button class="profile-main" on:click={toggleUserMenu}>
          <div class="avatar">{username[0]?.toUpperCase() || 'U'}</div>
          <div class="user-info">
            <div class="username">{username}</div>
            <div class="status">{$t('membership_gold')}</div>
          </div>
        </button>
        
        <button 
          class="settings-btn {activeView === 'settings' ? 'active' : ''}" 
          on:click={() => handleNavClick('settings')}
          title="Settings"
        >
          <Settings size={18} />
        </button>
      </div>
    </div>
  </div>
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
    font: var(--m3-headline-medium);
    height: 100%;
    line-height: 1;
    font-weight: 900;
    color: var(--md-primary);
    letter-spacing: -0.8px;
    display: flex;
    align-items: center;
  }

  .mini-logo { display: none; }

  .bottom-actions {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 12px;
  }

  .add-btn {
    height: 48px;
    padding: 0 20px;
    border-radius: var(--radius-l);
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border: none;
    font: var(--m3-label-large);
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    cursor: pointer;
    transition: all 0.2s;
    box-shadow: var(--shadow-1);
    margin: 0 12px;
  }
  .add-btn:hover { 
    background: var(--md-secondary-container);
    box-shadow: var(--shadow-2);
    transform: translateY(-2px);
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
    border-radius: var(--radius-m);
    transition: background 0.2s;
  }
  .popup-item:hover {
    background: var(--md-surface-variant);
  }

  .user-profile {
    padding: 6px;
    background: var(--md-surface-variant);
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid var(--md-outline);
    transition: all 0.2s;
  }

  .profile-main {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 4px 10px;
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
  }

  .user-info { flex: 1; min-width: 0; }
  .username { font: var(--m3-label-large); font-weight: 700; color: var(--md-on-surface); text-overflow: ellipsis; white-space: nowrap; overflow: hidden; }
  .status { font: var(--m3-label-small); color: var(--md-on-surface-variant); opacity: 0.8; }

  .settings-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    width: 36px;
    height: 36px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }
  .settings-btn:hover {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
  }
  .settings-btn.active {
    background: var(--md-primary);
    color: var(--md-on-primary);
  }

  /* Compact Mode (Tablet & Mobile) */
  @media (max-width: 1000px) {
    .sidebar {
      padding: 16px 8px;
      width: 72px !important;
    }
    .full-logo, 
    .nav-item span, 
    .cat-item span, 
    .section-title-text,
    .section-header div span,
    .add-btn span, 
    .user-info, 
    .badge, 
    .count {
      display: none !important;
    }
    .divider {
      margin: 12px 10px;
    }
    .mini-logo {
      display: block !important;
      font-size: 24px;
    }
    .logo {
      padding: 10px;
      margin: 0 auto;
    }
    .nav-item, .cat-item, .add-btn {
      justify-content: center;
      padding: 12px 0;
    }
    .item-left {
      gap: 0;
      justify-content: center;
    }
    .bottom-actions {
      align-items: center;
      padding: 12px 0;
    }
    .add-btn {
      width: 44px;
      height: 44px;
      padding: 0;
      margin: 8px 0;
      border-radius: var(--radius-full);
    }
    .user-profile {
      display: flex;
      flex-direction: column-reverse;
      align-items: center;
      gap: 12px;
      padding: 16px 0;
      width: 100%;
      background: transparent;
      border: none;
      margin: 0;
      position: relative;
    }
    .user-popup {
      left: 64px;
      bottom: 20px;
      width: 180px;
      right: auto;
      box-shadow: 4px 4px 20px rgba(0,0,0,0.3);
    }
    .avatar {
      width: 36px;
      height: 36px;
      font-size: 16px;
      border-radius: var(--radius-full);
      box-shadow: 0 0 0 2px var(--md-outline);
    }
    .settings-btn {
      width: 44px;
      height: 44px;
      background: var(--md-surface-variant);
      border-radius: var(--radius-l);
      display: flex;
      align-items: center;
      justify-content: center;
    }
  }
</style>

