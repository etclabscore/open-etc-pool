import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { loadConfig } from './lib/config';

// Load runtime config before the first render so endpoints/labels are correct,
// but never block the UI on it — fall back to defaults on failure.
loadConfig().finally(() => {
  mount(App, { target: document.getElementById('app')! });
});
