<script>
  import { onMount } from 'svelte';
  import { fetchStats } from './api.js';
  import { BarChart2, PieChart, Activity, Layers, Rss } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';

  let stats = null;
  let loading = true;

  onMount(async () => {
    try {
      stats = await fetchStats();
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  });

  $: maxCategory = stats?.category_stats?.reduce((max, s) => Math.max(max, s.count), 0) || 1;
  $: maxFeed = stats?.feed_stats?.reduce((max, s) => Math.max(max, s.count), 0) || 1;
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <BarChart2 size={20} class="header-icon" />
      <h1 class="title">Insights & Statistics</h1>
    </div>
  </header>

  <!-- Sharp edge fix for unibody rounding -->
  <div class="corner-bg"></div>
  <div class="corner-arc"></div>

  <div class="content-scroll scroll-area">
    {#if loading}
      <div class="loading-state">
        <div class="spinner"></div>
        <p>Calculating your digital intake...</p>
      </div>
    {:else if stats}
      <div class="stats-grid" in:fade>
        <!-- Top Row: Overview Cards -->
        <div class="stat-card primary" in:fly={{ y: 20, delay: 100 }}>
          <div class="card-icon"><Layers size={24} /></div>
          <div class="card-info">
             <span class="value">{stats.total_entries.toLocaleString()}</span>
             <span class="label">Total Articles</span>
          </div>
        </div>

        <div class="stat-card accent" in:fly={{ y: 20, delay: 200 }}>
          <div class="card-icon"><Activity size={24} /></div>
           <div class="card-info">
             <span class="value">{stats.unread_entries.toLocaleString()}</span>
             <span class="label">Unread Items</span>
          </div>
        </div>

        <div class="stat-card tertiary" in:fly={{ y: 20, delay: 300 }}>
          <div class="card-icon"><Rss size={24} /></div>
           <div class="card-info">
             <span class="value">{stats.total_feeds.toLocaleString()}</span>
             <span class="label">Active Sources</span>
          </div>
        </div>
      </div>

      <div class="details-grid">
        <!-- Category Distribution -->
        <div class="box category-box" in:fly={{ x: -20, delay: 400 }}>
          <div class="box-header">
            <PieChart size={18} />
            <span>Volume by Category</span>
          </div>
          <div class="chart-list">
            {#each stats.category_stats || [] as cat}
              <div class="chart-row">
                <div class="row-meta">
                  <span class="row-name">{cat.name}</span>
                  <span class="row-value">{cat.count}</span>
                </div>
                <div class="progress-bg">
                  <div class="progress-fill" style="width: {(cat.count / maxCategory) * 100}%"></div>
                </div>
              </div>
            {/each}
          </div>
        </div>

        <!-- Top Sources -->
        <div class="box sources-box" in:fly={{ x: 20, delay: 500 }}>
          <div class="box-header">
            <Activity size={18} />
            <span>Top 10 Sources</span>
          </div>
          <div class="chart-list">
            {#each stats.feed_stats || [] as feed}
              <div class="chart-row">
                <div class="row-meta">
                  <span class="row-name">{feed.title}</span>
                  <span class="row-value">{feed.count}</span>
                </div>
                <div class="progress-bg secondary">
                  <div class="progress-fill" style="width: {(feed.count / maxFeed) * 100}%"></div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .view-container { 
    display: flex; 
    flex-direction: column; 
    height: 100vh; 
    background: var(--md-surface); 
    flex: 1; 
    width: 100%; 
    position: relative;
    overflow: hidden;
  }
  .header-bar { 
    height: var(--header-height); 
    padding: 0 40px; 
    display: flex; 
    align-items: center; 
    background: var(--md-surface); 
    gap: 48px; 
    box-sizing: border-box;
    position: relative;
    z-index: 20;
    border-bottom: 1px solid var(--md-outline);
  }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-icon { color: var(--md-primary); }
  .title { font: var(--m3-headline-small); font-weight: 700; color: var(--md-primary); }

  .corner-bg { 
    position: absolute; top: var(--header-height); left: 0; 
    width: 32px; height: 32px; background: var(--md-surface); z-index: 0; 
  }
  .corner-arc { 
    position: absolute; top: var(--header-height); left: 0; 
    width: 32px; height: 32px; border-top-left-radius: 32px; 
    border-top: 1px solid var(--md-outline); border-left: 1px solid var(--md-outline); 
    z-index: 20; pointer-events: none; 
  }

  .content-scroll { 
    padding: 40px; 
    flex: 1; 
    width: 100%; 
    box-sizing: border-box; 
    overflow-y: auto; 
    background: var(--md-background);
    border-top-left-radius: 32px;
    border-left: 1px solid var(--md-outline);
    position: relative;
    z-index: 10;
  }

  .loading-state {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    height: 60vh; color: var(--md-on-surface-variant); gap: 16px;
  }
  .spinner {
    width: 40px; height: 40px; border: 4px solid var(--md-surface-variant);
    border-top-color: var(--md-primary); border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .stats-grid { 
    display: grid; 
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); 
    gap: 24px; 
    margin-bottom: 40px; 
    max-width: 1200px;
  }

  .stat-card {
    padding: 24px;
    border-radius: 24px;
    display: flex;
    align-items: center;
    gap: 20px;
    border: 1px solid var(--md-outline);
    background: var(--md-surface);
    transition: transform 0.2s;
  }
  .stat-card:hover { transform: translateY(-4px); }

  .stat-card.primary { background: var(--md-primary-container); border-color: var(--md-primary); }
  .stat-card.primary .card-icon { background: var(--md-primary); color: var(--md-on-primary); }
  .stat-card.accent { background: var(--md-secondary-container); border-color: var(--md-secondary); }
  .stat-card.accent .card-icon { background: var(--md-secondary); color: var(--md-on-secondary); }
  .stat-card.tertiary { background: var(--md-surface-variant); border-color: var(--md-outline); }
  .stat-card.tertiary .card-icon { background: var(--md-on-surface-variant); color: var(--md-surface-variant); }

  .card-icon {
    width: 56px; height: 56px; border-radius: 16px;
    display: flex; align-items: center; justify-content: center;
  }

  .card-info { display: flex; flex-direction: column; }
  .card-info .value { font-size: 32px; font-weight: 800; color: var(--md-on-surface); line-height: 1; margin-bottom: 4px; }
  .card-info .label { font: var(--m3-label-large); font-weight: 700; opacity: 0.7; text-transform: uppercase; letter-spacing: 0.5px; }

  .details-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
    gap: 24px;
    max-width: 1200px;
  }

  .box {
    background: var(--md-surface);
    border: 1px solid var(--md-outline);
    border-radius: 24px;
    padding: 24px;
  }

  .box-header {
    display: flex; align-items: center; gap: 12px;
    font: var(--m3-title-medium); font-weight: 800; color: var(--md-primary);
    margin-bottom: 24px; text-transform: uppercase;
  }

  .chart-list { display: flex; flex-direction: column; gap: 16px; }

  .chart-row { display: flex; flex-direction: column; gap: 6px; }
  .row-meta { display: flex; justify-content: space-between; font: var(--m3-body-medium); font-weight: 700; color: var(--md-on-surface); }
  .row-name { opacity: 0.8; }
  .row-value { color: var(--md-primary); }

  .progress-bg { height: 8px; background: var(--md-surface-variant); border-radius: 4px; overflow: hidden; }
  .progress-fill { height: 100%; background: var(--md-primary); border-radius: 4px; transition: width 1s ease-out; }
  .progress-bg.secondary .progress-fill { background: var(--md-secondary); }

  @media (max-width: 600px) {
    .details-grid { grid-template-columns: 1fr; }
    .content-scroll { padding: 20px; }
  }
</style>
