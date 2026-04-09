<script>
  import { Inbox, Star, Globe, Settings } from 'lucide-svelte';
  import { t } from './i18n.js';

  export let activeView = 'inbox';
  export let totalUnread = 0;
  export let onNavClick = (viewId) => {};
  export let onToggleSources = () => {};

  const items = [
    { id: 'inbox', icon: Inbox, label: 'inbox' },
    { id: 'favorites', icon: Star, label: 'favorites' },
    { id: 'sources', icon: Globe, label: 'sources', action: true },
    { id: 'settings', icon: Settings, label: 'settings' }
  ];

  function handleClick(item) {
    if (item.action && item.id === 'sources') {
      onToggleSources();
    } else {
      onNavClick(item.id);
    }
  }
</script>

<nav class="mobile-nav">
  {#each items as item}
    <button 
      class="nav-tab {activeView === item.id ? 'active' : ''}" 
      on:click={() => handleClick(item)}
    >
      <div class="icon-wrapper">
        <svelte:component this={item.icon} size={24} strokeWidth={activeView === item.id ? 2.5 : 2} />
        {#if item.id === 'inbox' && totalUnread > 0}
          <span class="badge">{totalUnread}</span>
        {/if}
      </div>
      <span class="label">{$t(item.label)}</span>
      {#if activeView === item.id}
        <div class="active-indicator"></div>
      {/if}
    </button>
  {/each}
</nav>

<style>
  .mobile-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 72px;
    background: var(--md-surface);
    display: flex;
    justify-content: space-around;
    align-items: center;
    border-top: 1px solid var(--md-outline);
    z-index: 1000;
    padding-bottom: env(safe-area-inset-bottom);
    box-shadow: 0 -4px 20px rgba(0,0,0,0.1);
  }

  .nav-tab {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    gap: 4px;
    padding: 0;
    position: relative;
    cursor: pointer;
  }

  .nav-tab.active {
    color: var(--md-primary);
  }

  .icon-wrapper {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  }

  .nav-tab:active .icon-wrapper {
    transform: scale(0.9);
  }

  .label {
    font-size: 11px;
    font-weight: 600;
    opacity: 0.8;
  }

  .nav-tab.active .label {
    opacity: 1;
  }

  .badge {
    position: absolute;
    top: -6px;
    right: -10px;
    background: var(--md-primary);
    color: var(--md-on-primary);
    font-size: 10px;
    font-weight: 800;
    padding: 2px 6px;
    border-radius: 10px;
    min-width: 14px;
    text-align: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }

  .active-indicator {
    position: absolute;
    bottom: 12px;
    width: 32px;
    height: 3px;
    background: var(--md-primary);
    border-radius: 3px;
    display: none; /* We use scale and color instead, or can enable if desired */
  }
</style>
