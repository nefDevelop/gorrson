<script>
  import { fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import PostCard from './PostCard.svelte';
  import { Search, CheckCheck } from 'lucide-svelte';
  import { t } from './i18n.js';
  
  /** @type {Array<any>} */
  export let articles = [];
  
  /** @type {number} */
  export let selectedId;
  
  /** @type {(id: number) => void} */
  export let onSelect;

</script>

<section class="feed-list" transition:fly={{ x: -20, duration: 400, easing: cubicOut }}>
  <header class="header-bar">
    <div></div> <!-- Empty spacer to keep actions aligned -->
    <div class="actions">
      <button class="action-btn">
        <Search size={18} />
      </button>
      <button class="action-btn">
        <CheckCheck size={18} />
      </button>
    </div>
  </header>

  <!-- The Inverse Corner to flawlessly draw the curve behind the gap -->
  <div class="inverse-corner"></div>

  <div class="feed-content">
    <div class="articles-container scroll-area">
      {#each articles as item}
        <PostCard 
          article={{...item, selected: selectedId === item.id}} 
          onClick={onSelect} 
        />
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
    height: 1px; /* Changed to exactly 1px */
    background: var(--md-surface);
    z-index: 15;
  }

  .inverse-corner {
    position: absolute;
    top: var(--header-height);
    left: 0;
    width: 32px;
    height: 32px;
    background: var(--md-surface);
    border-bottom-right-radius: 32px;
    border-right: 1px solid var(--md-outline);
    border-bottom: 1px solid var(--md-outline);
    z-index: 5; /* Behind feed-content */
    box-sizing: border-box;
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
    gap: 8px;
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 8px;
    border-radius: var(--radius-m);
    transition: all 0.2s;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }


  .articles-container {
    flex: 1;
    padding: 24px 16px 24px 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  @media (max-width: 1000px) {
    .header-bar {
      padding-left: 12px;
      padding-right: 12px;
    }
  }

  @media (max-width: 768px) {
    .feed-list {
      width: 100% !important;
      border-right: none;
    }
  }
</style>
