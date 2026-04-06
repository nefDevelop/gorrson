<script>
  import { t, locale } from './i18n.js';
  import { Settings, Globe, Shield, Info, LogOut, Plus } from 'lucide-svelte';
  import Filters from './Filters.svelte';

  export let onLogout = () => {};
  export let feeds = [];

  let activeTab = 'general';

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };

  const username = localStorage.getItem('username') || "User";
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <Settings size={20} />
      <h1 class="title">{$t('settings')}</h1>
    </div>
    <div class="tabs">
      <button class="tab {activeTab === 'general' ? 'active' : ''}" on:click={() => activeTab = 'general'}>General</button>
      <button class="tab {activeTab === 'filters' ? 'active' : ''}" on:click={() => activeTab = 'filters'}>{$t('filters')}</button>
    </div>
  </header>

  <div class="content-scroll scroll-area">
    {#if activeTab === 'general'}
      <div class="settings-section">
        <div class="section-header">
          <Globe size={18} />
          <span>General</span>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">Language / Idioma</div>
            <div class="desc">Chose your preferred interface language.</div>
          </div>
          <button class="btn-toggle" on:click={toggleLanguage}>
            {$locale === 'en' ? 'English' : 'Español'}
          </button>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Shield size={18} />
          <span>Account</span>
        </div>
        <div class="setting-item">
          <div class="text">
            <div class="label">User Profile</div>
            <div class="desc">Logged in as <strong>{username}</strong></div>
          </div>
          <button class="btn-danger" on:click={onLogout}>
            <LogOut size={16} />
            <span>{$t('logout')}</span>
          </button>
        </div>
      </div>

      <div class="settings-section">
        <div class="section-header">
          <Info size={18} />
          <span>System</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <span class="key">Version</span>
            <span class="value">v1.2.0-stable</span>
          </div>
          <div class="info-item">
            <span class="key">Engine</span>
            <span class="value">Go 1.22 + SQLite FTS5</span>
          </div>
          <div class="info-item">
            <span class="key">Architecture</span>
            <span class="value">Multi-tenant (Isolated databases)</span>
          </div>
        </div>
      </div>
    {:else}
      <div class="filters-view-embedded">
        <Filters {feeds} />
      </div>
    {/if}
  </div>
</div>

<style>
  .view-container { display: flex; flex-direction: column; height: 100vh; background: var(--md-background); flex: 1; width: 100%; }
  .header-bar { height: var(--header-height); padding: 0 24px; display: flex; align-items: center; border-bottom: 1px solid var(--md-outline); background: var(--md-surface); gap: 40px; }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .title { font: var(--m3-headline-small); font-weight: 700; color: var(--md-on-surface); }

  .tabs { display: flex; gap: 4px; height: 100%; align-items: center; }
  .tab { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    font: var(--m3-label-large); 
    padding: 0 16px; 
    height: 100%; 
    cursor: pointer; 
    border-bottom: 3px solid transparent; 
    transition: all 0.2s;
    opacity: 0.7;
  }
  .tab:hover { opacity: 1; color: var(--md-on-surface); }
  .tab.active { 
    border-bottom-color: var(--md-primary); 
    color: var(--md-primary); 
    font-weight: 700;
    opacity: 1;
  }

  .content-scroll { padding: 32px; flex: 1; width: 100%; box-sizing: border-box; overflow-y: auto; }
  
  .settings-section { margin-bottom: 40px; }
  .section-header { display: flex; align-items: center; gap: 10px; font: var(--m3-label-large); color: var(--md-primary); margin-bottom: 20px; opacity: 0.9; }

  .setting-item {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 16px 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .text .label { font: var(--m3-title-medium); color: var(--md-on-surface); margin-bottom: 4px; }
  .text .desc { font: var(--m3-body-small); color: var(--md-on-surface-variant); }

  .btn-toggle {
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    border: 1px solid var(--md-primary);
    padding: 8px 16px;
    border-radius: var(--radius-s);
    font: var(--m3-label-large);
    cursor: pointer;
    font-weight: 700;
  }

  .btn-danger {
    background: #ff525222;
    color: #ff5252;
    border: 1px solid #ff525244;
    padding: 8px 16px;
    border-radius: var(--radius-s);
    font: var(--m3-label-large);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 700;
  }

  .info-grid {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 20px;
    display: grid;
    gap: 12px;
  }

  .info-item { display: flex; justify-content: space-between; font: var(--m3-body-medium); }
  .info-item .key { color: var(--md-on-surface-variant); }
  .info-item .value { color: var(--md-on-surface); font-weight: 600; font-family: var(--font-mono); font-size: 13px; }

  .filters-view-embedded {
    margin: -32px; 
    height: calc(100vh - var(--header-height));
  }
</style>
