<script>
  import { 
    Plus, 
    Inbox, 
    Star, 
    Bookmark, 
    Archive, 
    Settings, 
    Hash,
    LogOut
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

  /** @type {string} */
  export let username = localStorage.getItem('username') || "User";

  export let onViewChange = (viewId) => {};
  export let onAddFeed = () => {};
  export let activeView = 'inbox';

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
  <div class="logo">
    <span class="full-logo">GORRSON</span>
    <span class="mini-logo">G</span>
  </div>
  
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

  <div class="section-title">{$t('categories')}</div>
  <nav class="categories-group">
    {#each categories as cat}
      <button 
        class="cat-item {activeView === 'cat:'+cat.id ? 'active' : ''}"
        on:click={() => handleNavClick('cat:'+cat.id)}
      >
        <div class="item-left">
          <Hash size={14} opacity={0.5} />
          <span>{$t(cat.id)}</span>
        </div>
        <span class="count">{cat.count}</span>
      </button>
    {/each}
  </nav>

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
    border-right: 1px solid var(--md-surface-variant);
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 16px;
    gap: 8px;
    transition: all var(--layout-transition);
  }

  .logo {
    padding: 20px;
    font: var(--m3-headline-medium);
    font-weight: 800;
    color: var(--md-on-surface);
    letter-spacing: -0.5px;
    display: flex;
    justify-content: center;
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
    padding: 12px 16px;
    border-radius: var(--radius-m);
    background: var(--md-surface-variant);
    color: var(--md-on-surface);
    border: 1px solid var(--md-outline);
    font: var(--m3-label-large);
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    cursor: pointer;
    transition: all 0.2s;
  }
  .add-btn:hover { 
    background: var(--md-secondary-container);
    border-color: var(--md-primary);
    transform: translateY(-1px);
  }

  .nav-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px;
  }

  .nav-item {
    padding: 8px 16px;
    border-radius: var(--radius-m);
    color: var(--md-on-surface-variant);
    font: var(--m3-body-medium);
    font-weight: 600;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: all 0.2s;
    background: transparent;
    border: none;
    width: 100%;
    outline: none;
  }

  .nav-item:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .nav-item.active { 
    background: var(--md-primary-container); 
    color: var(--md-on-primary-container); 
    font-weight: 700;
  }

  .item-left { display: flex; align-items: center; gap: 12px; }

  .badge {
    color: var(--md-primary);
    font: var(--m3-label-small);
    font-weight: 800;
  }

  .divider { height: 1px; background: var(--md-outline); opacity: 0.3; margin: 12px 20px; }

  .section-title {
    font: var(--m3-label-small);
    color: var(--md-on-surface-variant);
    text-transform: uppercase;
    padding: 12px 20px 8px 20px;
    opacity: 0.7;
    letter-spacing: 0.5px;
  }

  .categories-group { display: flex; flex-direction: column; gap: 2px; padding: 4px 12px; }
  .categories-group { 
    display: flex; 
    flex-direction: column; 
    gap: 2px; 
    padding: 0 12px; 
  }

  .cat-item {
    padding: 6px 16px;
    font: var(--m3-body-small);
    color: var(--md-on-surface-variant);
    background: transparent;
    border: none;
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
    border-radius: var(--radius-s);
    transition: all 0.2s;
  }
  .cat-item:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .cat-item.active { background: var(--md-primary-container); color: var(--md-on-primary-container); font-weight: 700; }
  
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
    border-radius: var(--radius-l);
    box-shadow: 0 4px 20px rgba(0,0,0,0.15), 0 8px 40px rgba(0,0,0,0.1);
    padding: 8px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow: hidden;
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
    .section-title, 
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

