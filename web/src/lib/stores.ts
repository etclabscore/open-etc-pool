import { createPoller } from './poller';
import { getStats } from './api';

// Shared pool-stats poll, consumed by the navbar (immature badge) and the home
// page. Started once for the app's lifetime in App.svelte.
export const statsPoller = createPoller(getStats);
