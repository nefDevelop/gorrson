<script>
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import PostCard from './PostCard.svelte';
  import { Search, CheckCheck, PanelRightClose, PanelRightOpen, Inbox, X } from 'lucide-svelte';
  import { t } from './i18n.js';
  import { fade, slide } from 'svelte/transition';
  
  /** @type {Array<any>} */
  export let articles = [];
  
  /** @type {number} */
  export let selectedId;
  
  export let isExpanded = false;
  export let onToggleExpand = () => {};

  /** @type {(id: number) => void} */
  export let onSelect;

  /** @type {(id: string) => void} */
  export let onToggleFavorite;

  /** @type {(article: any) => void} */
  export let onShare;

  export let searchQuery = "";
  let showSearch = false;
  let searchInput;

  function toggleSearch() {
    showSearch = !showSearch;
    if (!showSearch) searchQuery = "";
    else {
      setTimeout(() => searchInput?.focus(), 50);
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && showSearch) toggleSearch();
  }

</script>

<section class="feed-list {isExpanded ? 'expanded' : ''} {articles.length === 0 ? 'is-empty' : ''}" transition:fly={{ x: -20, duration: 400, easing: cubicOut }}>
  <header class="header-bar">
    <div></div> <!-- Empty spacer to keep actions aligned -->
    <div class="actions">
      <button class="action-btn {showSearch ? 'active' : ''}" on:click={toggleSearch} title="Search feeds">
        {#if showSearch}
          <X size={18} />
        {:else}
          <Search size={18} />
        {/if}
      </button>
      <button class="action-btn" title="Mark visible as read">
        <CheckCheck size={18} />
      </button>
      <button class="action-btn expand-btn" on:click={onToggleExpand} title={isExpanded ? 'Show reader panel' : 'Expand feed list'}>
        {#if isExpanded}
          <PanelRightOpen size={18} />
        {:else}
          <PanelRightClose size={18} />
        {/if}
      </button>
    </div>
  </header>

  {#if showSearch}
    <div class="floating-search" in:fly={{ y: -10, duration: 300 }} out:fly={{ y: -10, duration: 200 }}>
      <span class="search-icon-inner">
        <Search size={16} />
      </span>
      <input 
        bind:value={searchQuery} 
        bind:this={searchInput}
        placeholder={$t('search') || 'Search...'} 
        on:keydown={handleKeydown}
        class="search-input"
      />
      {#if searchQuery}
        <button class="clear-btn" on:click={() => searchQuery = ""}>
          <X size={14} />
        </button>
      {/if}
    </div>
  {/if}

  <!-- Solid surface background for the gap -->
  <div class="corner-bg"></div>
  
  <!-- Overlay to draw the perfectly curved sharp line -->
  <div class="corner-arc"></div>

  <div class="feed-content">
    <div class="articles-container scroll-area" class:search-open={showSearch}>
      {#each articles as item}
          <PostCard 
            article={{...item, selected: selectedId === item.id}} 
            onClick={onSelect} 
            {onToggleFavorite}
            onShare={() => onShare(item)}
          />
      {:else}
        <div class="empty-state" transition:fly={{ y: 20, duration: 400 }}>
          <div class="empty-icon">
            <div class="icon-circle">
              <Inbox size={48} strokeWidth={1.5} />
            </div>
          </div>
          <h3>{$t('all_caught_up') || 'All Caught Up'}</h3>
          <p>{$t('no_articles_desc') || 'There are no articles to display in this view right now.'}</p>
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .feed-list {
    position: relative;
    width: var(--feed-width);
    background: var(--md-background);
    display: flex;
    flex-direction: column;
    height: 100vh;
    transition: all var(--layout-transition);
  }
  .feed-list.expanded {
    width: calc(100vw - var(--sidebar-width));
    flex-shrink: 0;
  }
  .feed-list.is-empty {
    flex: 1;
    width: auto;
  }

  .header-bar { 
    display: flex; 
    justify-content: space-between; 
    align-items: center; 
    padding: 0 24px; 
    border-bottom: 1px solid var(--md-outline); 
    border-right: 1px solid var(--md-outline); 
    background: var(--md-surface);
    box-sizing: border-box; 
    height: var(--header-height); 
    min-height: var(--header-height); 
    max-height: var(--header-height);
    border-bottom-right-radius: 32px; 
    position: relative;
    z-index: 10;
  }

  /* Eraser for the straight line crossing the curved gap */
  .header-bar::before {
    content: "";
    position: absolute;
    left: 0px; 
    bottom: -1px; 
    width: 32px; 
    height: 2px;
    background: var(--md-surface);
    z-index: 15;
  }

  /* Solid background to color the gap outside the feed-content radius */
  .corner-bg {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    background: var(--md-surface);
    z-index: 0;
  }

  /* Overlaid pure arc to draw the crisp 1px unibody curve without clipping */
  .corner-arc {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    border-top-left-radius: 32px;
    border-top: 1px solid var(--md-outline);
    border-left: 1px solid var(--md-outline);
    z-index: 20; /* Keep stroke sharp and unclipped */
    pointer-events: none;
  }

  .feed-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
  }


  .actions {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-full);
    transition: all 0.2s;
    flex-shrink: 0;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }


  .articles-container {
    flex: 1;
    padding: 24px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    transition: padding-top 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .articles-container.search-open {
    padding-top: 80px;
  }

  .empty-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    text-align: center;
    color: var(--md-on-surface-variant);
  }

  .empty-icon {
    margin-bottom: 24px;
  }

  .icon-circle {
    width: 80px;
    height: 80px;
    border-radius: 50%;
    background: var(--md-surface-variant);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--md-primary);
  }

  .empty-state h3 {
    margin: 8px 0;
    color: var(--md-on-surface);
    font: var(--m3-headline-small);
    font-weight: 600;
  }

  .empty-state p {
    font: var(--m3-body-medium);
    max-width: 240px;
    margin: 0;
    opacity: 0.7;
    line-height: 1.5;
  }

  @media (max-width: 1000px) {
    .header-bar {
      padding-left: 12px;
      padding-right: 12px;
    }
    .feed-list {
      width: 100% !important;
      border-right: none;
    }
    .expand-btn {
      display: none !important;
    }
  }
  .action-btn.active {
    color: var(--md-primary);
    background: var(--md-primary-container);
  }

  .floating-search {
    position: absolute;
    top: calc(var(--header-height) + 12px);
    left: 24px;
    right: 24px;
    z-index: 100;
    display: flex;
    align-items: center;
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-full);
    padding: 0 16px;
    height: 44px;
    box-shadow: 0 4px 20px rgba(0,0,0,0.15);
    gap: 12px;
  }

  .search-icon-inner {
    color: var(--md-primary);
    opacity: 0.7;
  }

  .search-input {
    background: transparent;
    border: none;
    color: var(--md-on-surface);
    font: var(--m3-body-large);
    outline: none;
    flex: 1;
  }

  .clear-btn {
    background: var(--md-surface-variant);
    border: none;
    color: var(--md-on-surface-variant);
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
  }

  @media (max-width: 600px) {
    .floating-search {
      left: 12px;
      right: 12px;
    }
  }
</style>
