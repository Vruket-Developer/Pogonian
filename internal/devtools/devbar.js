/**
 * Pogonian In-Page DevBar & Live Reload Client
 * Injected automatically during development.
 */
(() => {
    'use strict';

    if (window.__pogn_devbar_loaded) return;
    window.__pogn_devbar_loaded = true;

    function createDevBar() {
        if (document.getElementById('pogn-devbar')) return;

        const bar = document.createElement('div');
        bar.id = 'pogn-devbar';
        bar.innerHTML = `
            <style>
                #pogn-devbar {
                    position: fixed;
                    bottom: 12px;
                    right: 16px;
                    z-index: 999999;
                    display: flex;
                    align-items: center;
                    gap: 10px;
                    background: rgba(10, 12, 16, 0.94);
                    backdrop-filter: blur(16px);
                    border: 1px solid rgba(56, 189, 248, 0.3);
                    border-radius: 9999px;
                    padding: 6px 14px;
                    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                    font-size: 12px;
                    color: #f0f3f6;
                    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
                    transition: all 0.2s ease;
                }
                #pogn-devbar:hover {
                    border-color: #38bdf8;
                    box-shadow: 0 12px 36px rgba(56, 189, 248, 0.25);
                }
                .pogn-dev-badge {
                    background: linear-gradient(135deg, #38bdf8, #818cf8);
                    color: #fff;
                    font-weight: 800;
                    padding: 2px 6px;
                    border-radius: 4px;
                    font-size: 10px;
                }
                .pogn-dev-metric {
                    color: #38bdf8;
                    font-weight: 600;
                }
                .pogn-dev-link {
                    color: #9aa5b5;
                    text-decoration: none;
                    display: flex;
                    align-items: center;
                    gap: 4px;
                    transition: color 0.15s ease;
                }
                .pogn-dev-link:hover {
                    color: #38bdf8;
                }
                .pogn-live-indicator {
                    width: 7px;
                    height: 7px;
                    border-radius: 50%;
                    background: #10b981;
                    box-shadow: 0 0 8px #10b981;
                    display: inline-block;
                }
                .pogn-live-indicator.connecting {
                    background: #f59e0b;
                    box-shadow: 0 0 8px #f59e0b;
                }
            </style>
            <span class="pogn-live-indicator" id="pogn-live-status" title="Live Reload Active"></span>
            <span class="pogn-dev-badge">DEV</span>
            <span>Route: <strong class="pogn-dev-metric">${window.location.pathname}</strong></span>
            <span>TTFB: <strong id="pogn-ttfb" class="pogn-dev-metric">~1ms</strong></span>
            <span style="color: rgba(255,255,255,0.2);">|</span>
            <a href="/swagger" target="_blank" class="pogn-dev-link" title="Open Interactive Swagger UI & OpenAPI Specification">
                <span>📚 Swagger</span>
            </a>
            <span style="color: rgba(255,255,255,0.2);">|</span>
            <a href="/__pogn/devtools" target="_blank" class="pogn-dev-link" title="Open Pogonian Architecture DevTools">
                <span>⚡ DevTools</span>
            </a>
        `;
        document.body.appendChild(bar);
    }

    // Connect to Server-Sent Events for Live Reload & Fast Refresh
    function setupLiveReload() {
        let evtSource = null;
        let reconnectDelay = 1000;

        function connect() {
            const statusDot = document.getElementById('pogn-live-status');
            if (statusDot) statusDot.className = 'pogn-live-indicator connecting';

            try {
                evtSource = new EventSource('/__pogn/live-reload');

                evtSource.onopen = () => {
                    if (statusDot) statusDot.className = 'pogn-live-indicator';
                    reconnectDelay = 1000;
                };

                evtSource.onmessage = (event) => {
                    const data = event.data;
                    if (data === 'reload' || data === 'html-update' || data === 'go-reloaded') {
                        // Reload page
                        window.location.reload();
                    } else if (data === 'css-update') {
                        // Hot swap CSS files without full page reload
                        hotReloadCSS();
                    }
                };

                evtSource.onerror = () => {
                    if (statusDot) statusDot.className = 'pogn-live-indicator connecting';
                    if (evtSource) {
                        evtSource.close();
                        evtSource = null;
                    }
                    setTimeout(connect, reconnectDelay);
                    reconnectDelay = Math.min(reconnectDelay * 1.5, 5000);
                };
            } catch (e) {
                setTimeout(connect, reconnectDelay);
            }
        }

        function hotReloadCSS() {
            const links = document.querySelectorAll('link[rel="stylesheet"]');
            links.forEach(link => {
                const href = link.getAttribute('href');
                if (href && !href.startsWith('http://') && !href.startsWith('https://')) {
                    const url = new URL(link.href);
                    url.searchParams.set('_pogn_t', Date.now());
                    link.href = url.toString();
                }
            });
        }

        connect();
    }

    // Update TTFB and route on soft navigation
    document.addEventListener('pogonian:navigated', (e) => {
        const metric = document.querySelector('#pogn-devbar strong.pogn-dev-metric');
        if (metric) {
            metric.textContent = window.location.pathname;
        }
    });

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            createDevBar();
            setupLiveReload();
        });
    } else {
        createDevBar();
        setupLiveReload();
    }
})();
