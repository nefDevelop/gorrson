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

  const themes = {
    lavender: {
      light: { primary: '#6750A4', secondary: '#625B71', tertiary: '#7D5260', container: '#EADDFF', onPrimary: '#FFFFFF', onContainer: '#21005D', surface: '#FEF7FF', onSurface: '#1D1B20', background: '#FFFFFF', variant: '#E7E0EB', outline: '#79747E' },
      dark: { primary: '#D0BCFF', secondary: '#CCC2DC', tertiary: '#EFB8C8', container: '#4F378B', onPrimary: '#381E72', onContainer: '#EADDFF', surface: '#1C1B1F', onSurface: '#E6E1E5', background: '#141218', variant: '#49454F', outline: '#938F99' }
    },
    midnight: {
      light: { primary: '#0061A4', secondary: '#535F70', tertiary: '#6B5778', container: '#D1E4FF', onPrimary: '#FFFFFF', onContainer: '#001D36', surface: '#F8F9FF', onSurface: '#191C20', background: '#FFFFFF', variant: '#DFE3EB', outline: '#73777F' },
      dark: { primary: '#9ECAFF', secondary: '#BBC7DB', tertiary: '#D6BEE4', container: '#00497D', onPrimary: '#003258', onContainer: '#D1E4FF', surface: '#111418', onSurface: '#E2E2E6', background: '#0B1015', variant: '#43474E', outline: '#8D9199' }
    },
    forest: {
      light: { primary: '#386A20', secondary: '#55624C', tertiary: '#386567', container: '#B7F397', onPrimary: '#FFFFFF', onContainer: '#042100', surface: '#F7FBF1', onSurface: '#191C19', background: '#FFFFFF', variant: '#DFE4D6', outline: '#72796F' },
      dark: { primary: '#9CD67D', secondary: '#BDCBB0', tertiary: '#A0CFD0', container: '#205107', onPrimary: '#0C3900', onContainer: '#B7F397', surface: '#1A1C19', onSurface: '#E2E3DD', background: '#111411', variant: '#424940', outline: '#8C9388' }
    },
    ruby: {
      light: { primary: '#BC1624', secondary: '#775652', tertiary: '#715B29', container: '#FFDAD6', onPrimary: '#FFFFFF', onContainer: '#410002', surface: '#FFF8F7', onSurface: '#221919', background: '#FFFFFF', variant: '#F5DDDA', outline: '#857371' },
      dark: { primary: '#FFB3AD', secondary: '#E7BDB7', tertiary: '#E0C28C', container: '#93000E', onPrimary: '#680007', onContainer: '#FFDAD6', surface: '#1A1110', onSurface: '#F1DEDC', background: '#211A19', variant: '#534341', outline: '#9F8C8A' }
    }
  };

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

    // Seed mappings
    const primary = isDark ? hslToHex(h, s, 80) : hex;
    const secondary = hslToHex((h + 10) % 360, Math.min(s, 30), isDark ? 70 : 50);
    const tertiary = hslToHex((h + 120) % 360, Math.min(s + 10, 70), isDark ? 80 : 50);
    const neutral = hslToHex(h, 8, isDark ? 10 : 98);
    const neutralVariant = hslToHex(h, 12, isDark ? 25 : 90);

    if (!isDark) {
      return {
        primary: hex,
        onPrimary: '#FFFFFF',
        container: hslToHex(h, Math.max(s - 30, 10), 92),
        onContainer: hslToHex(h, s, 10),
        secondary: secondary,
        tertiary: tertiary,
        surface: hslToHex(h, 6, 98),
        onSurface: '#1C1B1F',
        onSurfaceVariant: '#49454F',
        background: '#FFFFFF',
        variant: neutralVariant,
        outline: hslToHex(h, 12, 50)
      };
    } else {
      return {
        primary: primary,
        onPrimary: hslToHex(h, s, 20),
        container: hslToHex(h, s, 30),
        onContainer: hslToHex(h, s, 90),
        secondary: secondary,
        tertiary: tertiary,
        surface: hslToHex(h, 10, 12),
        onSurface: '#E6E1E5',
        onSurfaceVariant: '#CAC4D0',
        background: neutral,
        variant: neutralVariant,
        outline: hslToHex(h, 12, 60)
      };
    }
  }

  function applyAppearance(themeId, modeId, seed) {
    let mode = modeId;
    if (mode === 'system') {
      mode = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    let theme;
    if (themeId === 'custom') {
      theme = generateTonal(seed, mode);
    } else {
      theme = themes[themeId][mode];
    }
    
    const root = document.documentElement;
    root.style.setProperty('--md-primary', theme.primary);
    root.style.setProperty('--md-on-primary', theme.onPrimary);
    root.style.setProperty('--md-primary-container', theme.container);
    root.style.setProperty('--md-on-primary-container', theme.onContainer);
    
    root.style.setProperty('--md-secondary', theme.secondary);
    root.style.setProperty('--md-tertiary', theme.tertiary);
    
    root.style.setProperty('--md-secondary-container', mode === 'light' ? theme.variant : theme.container);
    root.style.setProperty('--md-on-secondary-container', theme.onSurface);

    root.style.setProperty('--md-surface', theme.surface);
    root.style.setProperty('--md-on-surface', theme.onSurface);
    root.style.setProperty('--md-surface-variant', theme.variant);
    root.style.setProperty('--md-on-surface-variant', theme.onSurfaceVariant || (mode === 'light' ? '#49454F' : '#CAC4D0'));
    root.style.setProperty('--md-background', theme.background);
    root.style.setProperty('--md-outline', theme.outline);

    localStorage.setItem('gorrson-theme', themeId);
    localStorage.setItem('gorrson-mode', modeId);
    localStorage.setItem('gorrson-custom-color', customColor);
  }

  $: if (typeof document !== 'undefined') {
     applyAppearance(currentTheme, currentMode, customColor);
  }

  // Auto-switch to custom if user tinkers with the color picker
  let prevColor = customColor;
  $: if (customColor !== prevColor) {
    currentTheme = 'custom';
    prevColor = customColor;
  }

  const toggleLanguage = () => {
    locale.update(l => l === 'en' ? 'es' : 'en');
  };

  const username = localStorage.getItem('username') || "User";
</script>

<div class="view-container">
  <header class="header-bar">
    <div class="header-left">
      <SettingsIcon size={20} />
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
          <Palette size={18} />
          <span>Appearance</span>
        </div>
        
        <div class="setting-item-column">
          <div class="label">Color Palette</div>
          <div class="palette-grid">
            {#each Object.entries(themes) as [id, theme]}
              <button 
                class="palette-card {currentTheme === id ? 'active' : ''}" 
                on:click={() => currentTheme = id}
              >
                <div class="palette-colors custom-preview">
                  <div class="color-dot" style="background: {theme.dark.primary}"></div>
                  <div class="color-dot" style="background: {theme.dark.secondary}"></div>
                  <div class="color-dot" style="background: {theme.dark.tertiary}"></div>
                  <div class="color-dot" style="background: {theme.dark.background}"></div>
                </div>
                <span>{id.charAt(0).toUpperCase() + id.slice(1)}</span>
              </button>
            {/each}
            
            <button 
              class="palette-card custom-card {currentTheme === 'custom' ? 'active' : ''}"
              on:click={() => currentTheme = 'custom'}
            >
              <div class="palette-colors custom-preview">
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').primary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').secondary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').tertiary}"></div>
                <div class="color-dot" style="background: {generateTonal(customColor, 'dark').background}"></div>
                <label class="picker-overlay">
                   <input type="color" bind:value={customColor} />
                </label>
              </div>
              <span>Custom Color</span>
            </button>
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

  .setting-item-column {
    background: var(--md-surface-variant);
    border: 1px solid var(--md-outline);
    border-radius: var(--radius-m);
    padding: 16px 20px;
    margin-bottom: 12px;
  }
  .setting-item-column .label { font: var(--m3-title-medium); color: var(--md-on-surface); margin-bottom: 16px; }

  .palette-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; }
  .palette-card { 
    background: var(--md-surface); 
    border: 1px solid var(--md-outline); 
    border-radius: var(--radius-m); 
    padding: 12px; 
    cursor: pointer; 
    display: flex; 
    flex-direction: column; 
    align-items: center; 
    gap: 8px;
    transition: all 0.2s;
    color: var(--md-on-surface-variant);
    font: var(--m3-label-medium);
  }
  .palette-card:hover { border-color: var(--md-primary); background: var(--md-primary-container); color: var(--md-on-primary-container); }
  .palette-card.active { border-color: var(--md-primary); background: var(--md-primary-container); color: var(--md-on-primary-container); border-width: 2px; }

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
  .custom-card { position: relative; cursor: pointer; }

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
</style>
