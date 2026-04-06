<script>
  import { stripHtml } from './utils.js';
  /** @type {{id: number, title: string, source: string, time: string, category: string, content?: string, summary?: string, selected?: boolean}} */
  export let article;
  
  /** @type {(id: number) => void} */
  export let onClick;

  $: summary = stripHtml(article.content || article.summary || "");
</script>

<button 
  class="post-card {article.selected ? 'selected' : ''}" 
  on:click={() => onClick(article.id)}
>
  <div class="title-row">
    <h3 class="title" class:active-title={article.selected}>{article.title}</h3>
    <span class="time">{article.time}</span>
  </div>

  <p class="summary">
    {summary || "No preview available..."}
  </p>
</button>

<style>
  .post-card {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: 24px;
    padding: 16px 20px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s cubic-bezier(0, 0, 0.2, 1);
    width: 100%;
    outline: none;
    display: block;
    text-align: left;
  }

  .post-card:hover {
    background: var(--md-secondary-container);
    border-color: var(--md-primary);
    transform: translateX(2px);
  }

  .post-card.selected {
    background: var(--md-primary-container);
    border-color: var(--md-primary);
    box-shadow: none; /* Adwaita 4 prefers flat surfaces */
  }

  .post-card.selected .title,
  .post-card.selected .summary,
  .post-card.selected .time {
    color: var(--md-on-primary-container);
    opacity: 1;
  }
  .title-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px; /* Espacio extra y generoso hacia la descripción */
  }

  .title {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 800;
    margin: 0;
    flex: 1;
    line-height: 1.35;
    color: var(--md-on-surface);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .time {
    font-family: var(--font-main);
    font-size: 11px;
    color: var(--md-on-surface-variant);
    opacity: 0.6;
    flex-shrink: 0;
    white-space: nowrap;
    margin-top: 4px;
    font-weight: 500;
  }

  .summary {
    font-family: var(--font-main);
    font-size: 13px;
    color: var(--md-on-surface-variant);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.5;
    opacity: 0.8;
  }
</style>
