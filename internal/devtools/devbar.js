/**
 * Pogonian In-Page DevBar (v0.1.0)
 * Injected automatically during development.
 */
(() => {
    'use strict';

    if (window.__pogn_devbar_loaded) return;
    window.__pogn_devbar_loaded = true;

    function createDevBar() {
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
                    gap: 12px;
                    background: rgba(10, 12, 16, 0.92);
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
                    color: #fff;
                }
            </style>
            <span class="pogn-dev-badge">POGONIAN DEV</span>
            <span>Route: <strong class="pogn-dev-metric">${window.location.pathname}</strong></span>
            <span>TTFB: <strong id="pogn-ttfb" class="pogn-dev-metric">~1ms</strong></span>
            <span style="color: rgba(255,255,255,0.2);">|</span>
            <a href="/__pogn/devtools" target="_blank" class="pogn-dev-link">
                <span>⚡ DevTools</span>
            </a>
        `;
        document.body.appendChild(bar);
    }

    // Update TTFB and route on soft navigation
    document.addEventListener('pogonian:navigated', (e) => {
        const metric = document.querySelector('#pogn-devbar strong.pogn-dev-metric');
        if (metric) {
            metric.textContent = window.location.pathname;
        }
    });

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', createDevBar);
    } else {
        createDevBar();
    }
})();
