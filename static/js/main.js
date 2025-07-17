// Auto-refresh functionality for DDNS Pilot
function setupAutoRefresh(enabled, intervalMinutes) {
    if (enabled && intervalMinutes > 0) {
        setTimeout(function() {
            window.location.reload();
        }, intervalMinutes * 60 * 1000);
    }
}

// Initialize auto-refresh based on server configuration
// This will be called from templates that need auto-refresh
function initAutoRefresh(config) {
    if (config && config.AutoUpdate && config.UpdateInterval) {
        setupAutoRefresh(true, config.UpdateInterval);
    }
} 