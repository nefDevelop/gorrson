<script>
  import { t, locale } from './i18n.js';
  import { Globe, Shield, Info, LogOut, Palette, Moon, Sun, Monitor, Settings as SettingsIcon } from 'lucide-svelte';
  import Filters from './Filters.svelte';
  import { onMount } from 'svelte';

  export let onLogout = () => {};
  export let feeds = [];

  let activeTab = 'general';
  
  // Theme state
  let currentTheme = localStorage.getItem('gorrson-theme') || 'lavender';
  let currentMode = localStorage.getItem('gorrson-mode') || 'dark';
  let customColor = localStorage.getItem('gorrson-custom-color') || '#6750A4';


  const modes = [
    { id: 'light', icon: Sun },
    { id: 'dark', icon: Moon },
    { id: 'system', icon: Monitor }
  ];

  function hexToHsl(hex) {
    let r = parseInt(hex.slice(1, 3), 16) / 255;
    let g = parseInt(hex.slice(3, 5), 16) / 255;
    let b = parseInt(hex.slice(5, 7), 16) / 255;
    let max = Math.max(r, g, b), min = Math.min(r, g, b);
    let h, s, l = (max + min) / 2;
    if (max === min) h = s = 0;
    else {
      let d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }
      h /= 6;
    }
    return [h * 360, s * 100, l * 100];
  }

  function hslToHex(h, s, l) {
    l /= 100;
    const a = s * Math.min(l, 1 - l) / 100;
    const f = n => {
      const k = (n + h / 30) % 12;
      const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
      return Math.round(255 * color).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
  }

  function generateTonal(hex, mode) {
    const [h, s, l] = hexToHsl(hex);
    const isDark = mode === 'dark';

    // MD3 Expressive: Vibrant palettes with higher chroma
    const primary = isDark ? hslToHex(h, Math.max(s, 50), 80) : hex;
    const secondary = hslToHex((h + 25) % 360, Math.max(s - 15, 30), isDark ? 70 : 40);
    const secondaryContainer = isDark 
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 25)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90);
    const onSecondaryContainer = isDark
      ? hslToHex((h + 25) % 360, Math.max(s - 15, 30), 90)
      : hslToHex((h + 25) % 360, Math.max(s - 15, 30), 10);
    const tertiary = hslToHex((h + 80) % 360, Math.max(s, 40), isDark ? 75 : 45);

    
    // Adwaita 4 Standard Surfaces (Tinted slightly with Seed Hue)
    const adwaita = {
      dark: { 
        bg: hslToHex(h, 5, 12), 
        surface: hslToHex(h, 6, 16), 
        card: hslToHex(h, 7, 20), 
        border: hslToHex(h, 10, 24) 
      },
      light: { 
        bg: '#FFFFFF', 
        surface: hslToHex(h, 10, 96), 
        card: '#FFFFFF', 
        border: hslToHex(h, 15, 88) 
      }
    };
    const currentA = isDark ? adwaita.dark : adwaita.light;

    if (!isDark) {
      return {
        primary: hex,
        onPrimary: '#FFFFFF',
        container: hslToHex(h, Math.max(s, 70), 92), // Very saturated for "Expressive"
        onContainer: hslToHex(h, s, 10),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 20, 10),
        onSurfaceVariant: hslToHex(h, 15, 40),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    } else {
      return {
        primary: primary,
        onPrimary: hslToHex(h, s, 15),
        container: hslToHex(h, Math.max(s, 60), 25), // Saturated highlight
        onContainer: hslToHex(h, Math.max(s, 40), 90),
        secondary: secondary,
        secondaryContainer: secondaryContainer,
        onSecondaryContainer: onSecondaryContainer,
        tertiary: tertiary,
        surface: currentA.surface,
        onSurface: hslToHex(h, 10, 95),
        onSurfaceVariant: hslToHex(h, 15, 75),
        background: currentA.bg,
        variant: currentA.card,
        outline: currentA.border
      };
    }
  }

  function applyAppearance(themeId, modeId, seed) {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    
    let mode = modeId;
    if (mode === 'system') {
      mode = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    // Since we only use Seed Color now
    const tonals = generateTonal(seed, mode);
    
    // MD3 Core Tokens
    root.style.setProperty('--md-primary', tonals.primary);
    root.style.setProperty('--md-on-primary', tonals.onPrimary);
    root.style.setProperty('--md-primary-container', tonals.container);
    root.style.setProperty('--md-on-primary-container', tonals.onContainer);
    
    root.style.setProperty('--md-secondary', tonals.secondary);
    root.style.setProperty('--md-secondary-container', tonals.secondaryContainer);
    root.style.setProperty('--md-on-secondary-container', tonals.onSecondaryContainer);
    root.style.setProperty('--md-tertiary', tonals.tertiary);

    // Adwaita + Tonal Alignment
    root.style.setProperty('--md-surface', tonals.surface);
    root.style.setProperty('--md-on-surface', tonals.onSurface);
    root.style.setProperty('--md-on-surface-variant', tonals.onSurfaceVariant);
    
    root.style.setProperty('--md-background', tonals.background);
    root.style.setProperty('--md-surface-variant', tonals.variant);
    root.style.setProperty('--md-outline', tonals.outline);
    
    // UI Effects
    root.style.setProperty('--shadow-color', mode === 'light' ? 'rgba(0,0,0,0.06)' : 'rgba(0,0,0,0.3)');

    localStorage.setItem('gorrson-theme', themeId);
    localStorage.setItem('gorrson-mode', modeId);
    localStorage.setItem('gorrson-custom-color', seed);
  }

  $: if (typeof document !== 'undefined') {
     applyAppearance('custom', currentMode, customColor);
  }

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };

  const username = localStorage.getItem('username') || "User";

  $: categoriesList = Array.from(new Set(feeds.map(f => f.category || 'General'))).sort();

  let editingCat = null;
  let newName = "";

  async function renameCategory(oldName) {
    if (!newName || newName === oldName) {
       editingCat = null;
       return;
    }
    const res = await fetch(`/categories/${encodeURIComponent(oldName)}`, {
      method: 'PUT',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ name: newName })
    });
    if (res.ok) {
      window.location.reload(); // Hard reload to refresh everything
    }
    editingCat = null;
  }
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <SettingsIcon size={20} />
      <h1 class="title">{$t('settings')}</h1>
    </div>
    <div class="tabs">
      <button class="tab {activeTab === 'general' ? 'active' : ''}" on:click={() => activeTab = 'general'}>General</button>
      <button class="tab {activeTab === 'categories' ? 'active' : ''}" on:click={() => activeTab = 'categories'}>{$t('categories')}</button>
      <button class="tab {activeTab === 'filters' ? 'active' : ''}" on:click={() => activeTab = 'filters'}>{$t('filters')}</button>
    </div>
  </header>

  <div class="content-scroll scroll-area">
    {#if activeTab === 'general'}
      <div class="settings-section">
        <div class="section-header">
          <Palette size={18} />
          <span>Appearance</span>
        </div>
        
        <div class="setting-item-column">
          <div class="label">Seed Color</div>
          <div class="desc" style="color: var(--md-on-surface-variant); font: var(--m3-body-small); margin-bottom: 20px;">
            Material Design 3 automatically generates harmonious primary, secondary, and tertiary palettes from a single seed color.
          </div>
          
          <div class="seed-color-container">
            <div class="seed-picker-card">
              <div class="palette-colors custom-preview large-preview">
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').primary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').secondary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').tertiary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').background}"></div>
                <label class="picker-overlay">
                   <input type="color" bind:value={customColor} />
                </label>
              </div>
              <span class="seed-label">Selected Seed: {customColor.toUpperCase()}</span>
            </div>
          </div>
        </div>

        <div class="setting-item">
          <div class="text">
            <div class="label">Theme Mode</div>
            <div class="desc">Switch between light and dark backgrounds.</div>
          </div>
          <div class="mode-selector">
            {#each modes as mode}
              <button 
                class="mode-btn {currentMode === mode.id ? 'active' : ''}" 
                on:click={() => currentMode = mode.id}
                title={mode.id}
              >
                <svelte:component this={mode.icon} size={18} />
              </button>
            {/each}
          </div>
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
    {:else if activeTab === 'categories'}
      <div class="settings-section">
        <div class="section-header">
          <Globe size={18} />
          <span>Manage Categories</span>
        </div>
        <div class="category-manager">
          {#each categoriesList as cat}
            <div class="cat-row">
              {#if editingCat === cat}
                <input 
                  class="edit-input"
                  bind:value={newName}
                  on:keydown={e => e.key === 'Enter' && renameCategory(cat)}
                  on:blur={() => editingCat = null}
                />
                <button class="btn-primary-small" on:click={() => renameCategory(cat)}>Save</button>
              {:else}
                <span class="cat-name">{cat}</span>
                <div class="row-actions">
                  <button class="btn-ghost" on:click={() => { editingCat = cat; newName = cat; }}>Rename</button>
                </div>
              {/if}
            </div>
          {/each}
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
  .header-bar { 
    height: var(--header-height); 
    padding: 0 24px; 
    display: flex; 
    align-items: center; 
    /* Removed border-bottom and border-left, managed by FeedList-Right */
    background: var(--md-surface); 
    gap: 40px; 
    box-sizing: border-box;
  }
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

  .setting-item-column {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 16px 20px;
    margin-bottom: 12px;
  }
  .setting-item-column .label { font: var(--m3-title-medium); color: var(--md-on-surface); margin-bottom: 16px; }

  .seed-color-container { display: flex; justify-content: center; padding: 10px 0; }
  .seed-picker-card { 
    background: var(--md-surface); 
    border: 2px solid var(--md-primary); 
    border-radius: var(--radius-m); 
    padding: 24px; 
    cursor: pointer; 
    display: flex; 
    flex-direction: column; 
    align-items: center; 
    gap: 16px;
    color: var(--md-on-surface);
    font: var(--m3-label-large);
    position: relative;
    transition: all 0.2s;
    width: 200px;
  }
  .seed-picker-card:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.2); }
  .custom-preview.large-preview { grid-template-columns: repeat(2, 32px); gap: 8px; }
  .custom-preview.large-preview .color-dot { width: 32px; height: 32px; border-radius: 8px; }
  .seed-label { font-family: var(--font-mono); font-size: 13px; opacity: 0.8; }

  .palette-colors { display: flex; gap: 4px; }
  .color-dot { width: 16px; height: 16px; border-radius: var(--radius-full); border: 1px solid rgba(255,255,255,0.1); }
  .custom-preview { position: relative; display: grid; grid-template-columns: repeat(2, 12px); gap: 2px; }
  .picker-overlay { position: absolute; top: 0; left: 0; width: 100%; height: 100%; cursor: pointer; }
  .picker-overlay input { 
    position: absolute; 
    top: 50%; left: 50%; 
    transform: translate(-50%, -50%);
    width: 40px; height: 40px; 
    cursor: pointer; 
    opacity: 0; 
  }

  .mode-selector { display: flex; background: var(--md-surface); padding: 4px; border-radius: var(--radius-full); border: 1px solid var(--md-outline); }
  .mode-btn { 
    background: transparent; 
    border: none; 
    color: var(--md-on-surface-variant); 
    width: 36px; 
    height: 36px; 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: all 0.2s;
  }
  .mode-btn.active { background: var(--md-primary); color: var(--md-on-primary); }
  .category-manager {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 16px;
  }
  .cat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    background: var(--md-surface);
    /* Removed border-bottom, respect single path */
    box-sizing: border-box;
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    transition: transform 0.2s;
  }
  .cat-row:hover {
    border-color: var(--md-primary);
  }
  .cat-name {
    font: var(--m3-body-large);
    font-weight: 700;
    color: var(--md-on-surface);
  }
  .edit-input {
    flex: 1;
    margin-right: 12px;
    padding: 8px 12px;
    border: 1px solid var(--md-primary);
    border-radius: var(--radius-s);
    background: var(--md-surface-variant);
    color: var(--md-on-surface);
    font: var(--m3-body-medium);
    outline: none;
  }
  .btn-primary-small {
    background: var(--md-primary);
    color: var(--md-on-primary);
    border: none;
    padding: 8px 16px;
    border-radius: var(--radius-full);
    font: var(--m3-label-medium);
    font-weight: 700;
    cursor: pointer;
  }
</style>
