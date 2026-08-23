/**
 * Pogonian Runtime (v0.1.0)
 * Zero-dependency SPA soft navigation, DOM patching, and async form interception.
 */
(() => {
    'use strict';

    console.log("%c[Pogonian]%c Soft navigation runtime active", "color: #38bdf8; font-weight: bold;", "color: inherit;");

    const ROOT_SELECTOR = '#pogn-root, main, [data-pogn-root]';

    function getRootElement(doc = document) {
        return doc.querySelector(ROOT_SELECTOR) || doc.body;
    }

    async function navigateTo(url, pushState = true) {
        try {
            document.body.classList.add('pogn-loading');
            const res = await fetch(url, {
                headers: {
                    'X-Pogonian-Request': 'true',
                    'Accept': 'text/html'
                }
            });

            if (!res.ok) {
                // Fallback to normal navigation if server returns error or non-200
                window.location.href = url;
                return;
            }

            const html = await res.text();
            const parser = new DOMParser();
            const newDoc = parser.parseFromString(html, 'text/html');

            // 1. Update Title
            if (newDoc.title) {
                document.title = newDoc.title;
            }

            // 2. Replace Root / Main content
            const currentRoot = getRootElement(document);
            const nextRoot = getRootElement(newDoc);

            if (currentRoot && nextRoot) {
                currentRoot.innerHTML = nextRoot.innerHTML;
            } else {
                document.body.innerHTML = newDoc.body.innerHTML;
            }

            // 3. Update active links in nav
            updateActiveLinks(url);

            // 4. Update History
            if (pushState) {
                window.history.pushState({ url }, '', url);
            }

            // 5. Scroll to top
            window.scrollTo({ top: 0, behavior: 'instant' });

            // 6. Dispatch custom event
            document.dispatchEvent(new CustomEvent('pogonian:navigated', { detail: { url } }));
        } catch (err) {
            console.error('[Pogonian] Soft navigation failed, falling back:', err);
            window.location.href = url;
        } finally {
            document.body.classList.remove('pogn-loading');
        }
    }

    function updateActiveLinks(currentUrl) {
        const path = new URL(currentUrl, window.location.origin).pathname;
        document.querySelectorAll('a[href]').forEach(link => {
            const href = link.getAttribute('href');
            if (href === path || (href !== '/' && path.startsWith(href))) {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });
    }

    // Intercept link clicks
    document.addEventListener('click', (e) => {
        const link = e.target.closest('a');
        if (!link) return;

        // Check if link should be handled natively
        const href = link.getAttribute('href');
        const target = link.getAttribute('target');
        const download = link.hasAttribute('download');
        const rel = link.getAttribute('rel');

        if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
        if (target && target !== '_self') return;
        if (download || (rel && rel.includes('external'))) return;
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

        // Origin check
        const targetUrl = new URL(href, window.location.origin);
        if (targetUrl.origin !== window.location.origin) return;

        e.preventDefault();
        navigateTo(targetUrl.href, true);
    });

    // Intercept form submissions
    document.addEventListener('submit', async (e) => {
        const form = e.target.closest('form');
        if (!form || form.dataset.pognNative !== undefined) return;

        const action = form.getAttribute('action') || window.location.href;
        const method = (form.getAttribute('method') || 'GET').toUpperCase();
        const targetUrl = new URL(action, window.location.origin);

        // Only handle same-origin forms
        if (targetUrl.origin !== window.location.origin) return;

        e.preventDefault();
        document.body.classList.add('pogn-loading');

        try {
            let res;
            if (method === 'GET') {
                const formData = new FormData(form);
                const params = new URLSearchParams(formData);
                targetUrl.search = params.toString();
                await navigateTo(targetUrl.href, true);
                return;
            } else {
                const formData = new FormData(form);
                res = await fetch(action, {
                    method: 'POST',
                    body: formData,
                    headers: {
                        'X-Pogonian-Request': 'true',
                        'Accept': 'text/html, application/json'
                    }
                });
            }

            if (res.redirected) {
                await navigateTo(res.url, true);
                return;
            }

            const contentType = res.headers.get('content-type') || '';
            if (contentType.includes('text/html')) {
                const html = await res.text();
                const parser = new DOMParser();
                const newDoc = parser.parseFromString(html, 'text/html');
                const currentRoot = getRootElement(document);
                const nextRoot = getRootElement(newDoc);
                if (currentRoot && nextRoot) {
                    currentRoot.innerHTML = nextRoot.innerHTML;
                }
                if (newDoc.title) document.title = newDoc.title;
            } else if (contentType.includes('application/json')) {
                const json = await res.json();
                document.dispatchEvent(new CustomEvent('pogonian:form-response', { detail: { data: json, status: res.status } }));
            }
        } catch (err) {
            console.error('[Pogonian] Form submission error:', err);
            form.submit(); // fallback
        } finally {
            document.body.classList.remove('pogn-loading');
        }
    });

    // Handle back / forward browser navigation
    window.addEventListener('popstate', (e) => {
        navigateTo(window.location.href, false);
    });

    // Initialize active links on load
    document.addEventListener('DOMContentLoaded', () => {
        updateActiveLinks(window.location.href);
    });
})();
