<script>
  import { fade } from 'svelte/transition';
  import { ArrowLeft, Link, Star, Share } from 'lucide-svelte';
  import { t } from './i18n.js';
  
  /** @type {{id: number, title: string, source: string, time: string, category: string, content?: string, img?: string}} */
  export let article;
</script>

<article class="post-viewer scroll-area">
  <div class="progress-bar"></div>
  
  <header class="header-bar article-header">
    <div class="header-left">
      <button class="action-btn">
        <ArrowLeft size={20} />
      </button>
      {#if article}
        <span class="header-source">{article.source}</span>
      {/if}
    </div>
    <div class="header-right">
      <button class="action-btn">
        <Link size={18} />
      </button>
      <button class="action-btn">
        <Star size={18} />
      </button>
      <button class="action-btn">
        <Share size={18} />
      </button>
    </div>
  </header>

  {#if article}
    <div in:fade={{ duration: 250 }} class="content-wrapper">
      {#if article.img}
        <div class="image-container">
          <img src={article.img} alt={article.title} />
        </div>
      {/if}
      
      <div class="meta">
        <span class="category">{$t(article.category.toLowerCase()) || article.category}</span>
        <span class="info">{article.source} • {article.time} {$t('read_time')}</span>
      </div>
      
      <h1 class="title">{article.title}</h1>
      
      <div class="body-text">
        {article.content}
      </div>
    </div>
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
  }

  .progress-bar {
    position: absolute;
    top: 0;
    left: 0;
    height: 2px;
    width: 45%; 
    background: var(--md-primary);
    z-index: 1001;
  }

  .article-header {
    background: var(--md-surface);
    border-bottom: 1px solid var(--md-outline);
    justify-content: space-between;
    height: var(--header-height);
  }

  .header-left, .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .header-source {
    font: var(--m3-title-medium);
    color: var(--md-on-surface);
    font-weight: 700;
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

  .content-wrapper {
    padding: 40px 48px;
    max-width: 900px;
    margin: 0 auto;
    width: 100%;
  }

  .image-container {
    width: 100%;
    height: 420px;
    border-radius: var(--radius-xl);
    overflow: hidden;
    margin-bottom: 32px;
    border: 1px solid var(--md-outline);
    background: var(--md-surface-variant);
  }

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .meta {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 24px;
  }

  .category {
    padding: 6px 16px;
    border-radius: var(--radius-full);
    background: var(--md-primary-container);
    color: var(--md-on-primary-container);
    font: var(--m3-label-large);
  }

  .info {
    color: var(--md-on-surface-variant);
    font: var(--m3-label-large);
    opacity: 0.8;
  }

  .title {
    font: var(--m3-display-small);
    line-height: 1.2;
    margin-bottom: 32px;
    color: var(--md-on-surface);
  }

  .body-text {
    font: var(--m3-body-large);
    line-height: 1.8;
    color: var(--md-on-surface);
    opacity: 0.9;
    white-space: pre-wrap;
  }

  .empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--md-on-surface-variant);
    font: var(--m3-body-large);
  }
</style>

