<script>
  import { 
    Plus, 
    Inbox, 
    Star, 
    Bookmark, 
    Archive, 
    Settings, 
    Hash,
    Languages,
    LogOut
  } from 'lucide-svelte';
  import { locale, t } from './i18n.js';

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
  
  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };
</script>

<aside class="sidebar">
  <div class="logo">GORRSON</div>
  
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

    <div class="user-profile">
      <div class="avatar">{username[0]?.toUpperCase() || 'U'}</div>
      <div class="user-info">
        <div class="username">{username}</div>
        <div class="status">{$t('membership_gold')}</div>
      </div>
      <div class="profile-actions">
        <button class="lang-btn" on:click={toggleLanguage} title="Change Language">
          <Languages size={18} />
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
  }

  .logo {
    padding: 20px;
    font: var(--m3-headline-medium);
    font-weight: 800;
    color: var(--md-on-surface);
    letter-spacing: -0.5px;
  }

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
  }
  .cat-item:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  
  .count { 
    background: var(--md-surface-variant);
    color: var(--md-on-surface-variant);
    padding: 2px 8px;
    border-radius: var(--radius-full);
    font: var(--m3-label-small);
    font-weight: 600;
  }
  .cat-item:hover .count {
    background: var(--md-secondary-container);
    color: var(--md-on-secondary-container);
  }

  .user-profile {
    padding: 12px;
    background: transparent;
    border-top: 1px solid var(--md-outline);
    display: flex;
    align-items: center;
    gap: 12px;
    border-radius: 0;
  }

  .avatar {
    width: 28px;
    height: 28px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    border-radius: var(--radius-m);
    display: flex;
    align-items: center;
    justify-content: center;
    font: var(--m3-label-medium);
    font-weight: 800;
  }

  .user-info { flex: 1; min-width: 0; }
  .username { font: var(--m3-label-large); font-weight: 700; color: var(--md-on-surface); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .status { font: var(--m3-label-small); color: var(--md-on-surface-variant); opacity: 0.8; }

  .profile-actions {
    display: flex;
    gap: 4px;
  }

  .settings-btn, .lang-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    font-size: 16px;
    padding: 6px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s;
  }

  .settings-btn:hover, .lang-btn:hover {
    background: rgba(0, 0, 0, 0.05);
    color: var(--md-on-surface);
  }
</style>

