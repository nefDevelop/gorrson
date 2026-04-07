<script>
  import { fade, fly } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { Link, Star, ExternalLink, ArrowLeft } from 'lucide-svelte';
  import { t } from './i18n.js';
  import { toggleFavorite } from './api.js';
  
  /** @type {any} */
  export let article;
  export let onBack = () => {};

  let progress = 0;
  let container;

  function handleScroll() {
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const total = scrollHeight - clientHeight;
    progress = total > 0 ? (scrollTop / total) * 100 : 0;
  }

  let lastHandledId = null;

  // Reset and auto-load full content when article changes
  $: if (article && article.id !== lastHandledId) {
    lastHandledId = article.id;
    progress = 0;
    if (container) container.scrollTop = 0;
    if (!article.is_full_content) {
      loadFullContent();
    }
  }

  async function handleFavorite() {
    if (!article) return;
    await toggleFavorite(article.id, !article.is_favorite);
    article = { ...article, is_favorite: !article.is_favorite };
  }

  let loadingFull = false;
  async function loadFullContent() {
    if (!article || loadingFull) return;
    loadingFull = true;
    try {
      const response = await fetch(`/news/${article.id}/content`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      if (response.ok) {
        const text = await response.text();
        article = { ...article, content: text, is_full_content: true };
      }
    } catch (e) {
      console.error(e);
    } finally {
      loadingFull = false;
    }
  }
</script>

<article 
  class="post-viewer scroll-area" 
  bind:this={container} 
  on:scroll={handleScroll}
  transition:fly={{ x: 100, duration: 400, easing: cubicOut }}
>
  <!-- Header removed per user request -->


  {#if article}
    {#key article.id}
      <div 
        in:fly={{ x: 100, duration: 600, delay: 100, easing: cubicOut }} 
        out:fade={{ duration: 200 }} 
        class="content-wrapper"
      >
        {#if article.image_url}
          <div class="image-container">
            <img src={article.image_url} alt={article.title} />
          </div>
        {/if}
        
        <div class="meta">
          <div class="meta-left">
            <button class="action-btn mobile-only back-btn" on:click={onBack} title="Back to list">
              <ArrowLeft size={20} />
            </button>
            <span class="category">{article.category || 'General'}</span>
            <span class="info">{article.source}</span>
          </div>
          <div class="meta-right">
            <span class="info time">{article.time}</span>
            <button class="action-btn {article.is_favorite ? 'active' : ''}" on:click={handleFavorite} title="Favorite">
              <Star size={18} fill={article.is_favorite ? 'currentColor' : 'none'} />
            </button>
            <a class="action-btn" href={article.url} target="_blank" rel="noopener" title="Open original">
              <ExternalLink size={18} />
            </a>
          </div>
        </div>
        
        <h1 class="title">{article.title}</h1>
        
        {#if loadingFull}
          <div class="loader-centered">
            <div class="mini-spinner"></div>
          </div>
        {/if}

        <div class="body-text prose">
          {@html article.content || article.summary || '<p>No content available.</p>'}
        </div>
      </div>
    {/key}
  {:else}
    <div class="empty-state">
      <p>{$t('select_article')}</p>
    </div>
  {/if}
</article>

<style>
  .post-viewer {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--md-background);
    position: relative;
    overflow-y: auto;
    overflow-x: hidden;
  }


  
  .loader-centered {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 24px;
    width: 100%;
    margin-bottom: 32px;
  }

  .mini-spinner {
    width: 28px;
    height: 28px;
    border: 3px solid var(--md-primary);
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .mobile-only {
    display: none !important;
  }

  @media (max-width: 1000px) {
    .mobile-only {
      display: flex !important;
    }
    .content-wrapper {
      padding: 24px 20px;
    }
    .title {
      font-size: 24px;
    }
  }

  .action-btn {
    background: transparent;
    border: none;
    color: var(--md-on-surface-variant);
    cursor: pointer;
    padding: 8px;
    border-radius: var(--radius-full); /* Redondeado perfecto MD3 */
    transition: all 0.2s;
    display: flex;
    align-items: center;
    text-decoration: none;
  }
  .action-btn:hover { background: var(--md-surface-variant); color: var(--md-on-surface); }
  .action-btn.active { color: var(--md-primary); }



  
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }


  .content-wrapper {
    padding: 40px 48px;
    max-width: 760px;
    margin: 0 auto;
    width: 100%;
  }

  .image-container {
    width: 100%;
    height: 360px;
    border-radius: var(--radius-xl);
    overflow: hidden;
    margin-bottom: 32px;
    border: 1px solid var(--md-outline);
    background: var(--md-surface-variant);
  }
  img { width: 100%; height: 100%; object-fit: cover; }

  .meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
    flex-wrap: wrap;
    gap: 12px;
  }
  .meta-left, .meta-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .meta-right {
    justify-content: flex-end;
  }
  .category {
    padding: 4px 14px;
    border-radius: var(--radius-full);
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    font: var(--m3-label-large);
    font-weight: 700;
  }
  .info {
    color: var(--md-on-surface-variant);
    font: var(--m3-label-large);
    opacity: 0.8;
  }

  .title {
    font-family: var(--font-display);
    font-size: 28px;
    font-weight: 800;
    line-height: 1.25;
    margin-bottom: 32px;
    color: var(--md-on-surface);
  }

  /* Prose: style the raw HTML from RSS feeds */
  .body-text :global(p) {
    font: var(--m3-body-large);
    line-height: 1.9;
    color: var(--md-on-surface);
    opacity: 0.9;
    margin-bottom: 20px;
  }
  .body-text :global(h1),
  .body-text :global(h2),
  .body-text :global(h3) {
    font-family: var(--font-display);
    font-weight: 700;
    color: var(--md-on-surface);
    margin: 32px 0 16px;
    line-height: 1.3;
  }
  .body-text :global(h2) { font-size: 22px; }
  .body-text :global(h3) { font-size: 18px; }
  .body-text :global(a) {
    color: var(--md-primary);
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .body-text :global(img) {
    max-width: 100%;
    border-radius: var(--radius-m);
    margin: 16px 0;
    height: auto;
    border: 1px solid var(--md-outline);
  }
  .body-text :global(blockquote) {
    border-left: 3px solid var(--md-primary);
    padding-left: 20px;
    margin: 20px 0;
    color: var(--md-on-surface-variant);
    font-style: italic;
  }
  .body-text :global(pre),
  .body-text :global(code) {
    font-family: var(--font-mono);
    background: var(--md-surface-variant);
    border-radius: var(--radius-s);
    font-size: 13px;
  }
  .body-text :global(pre) { padding: 16px; overflow-x: auto; }
  .body-text :global(code) { padding: 2px 6px; }
  .body-text :global(ul),
  .body-text :global(ol) {
    padding-left: 24px;
    margin-bottom: 16px;
    line-height: 1.9;
    color: var(--md-on-surface);
    opacity: 0.9;
  }
  .body-text :global(figure),
  .body-text :global(figcaption) {
    text-align: center;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-small);
  }
  .body-text :global(hr) {
    border: none;
    border-top: 1px solid var(--md-outline);
    margin: 32px 0;
  }

  .empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-large);
    opacity: 0.5;
  }
</style>
