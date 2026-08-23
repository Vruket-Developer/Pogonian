/**
 * Pogonian Runtime (v0.2.0)
 * Zero-dependency SPA soft navigation, DOM patching, head reconciliation, and multipart form interception.
 */
(() => {
    'use strict';

    if (window.__pogonian_runtime_loaded) return;
    window.__pogonian_runtime_loaded = true;

    console.log("%c[Pogonian]%c Architecture-aware soft navigation runtime active", "color: #38bdf8; font-weight: bold;", "color: inherit;");

    const ROOT_SELECTOR = '#pogn-root, main, [data-pogn-root]';

    const lifecycle = {
        beforeNavigate: [],
        beforeSwap: [],
        afterNavigate: []
    };

    window.Pogonian = {
        onBeforeNavigate: (fn) => lifecycle.beforeNavigate.push(fn),
        onBeforeSwap: (fn) => lifecycle.beforeSwap.push(fn),
        onAfterNavigate: (fn) => lifecycle.afterNavigate.push(fn),
        navigate: (url) => navigateTo(url, true)
    };

    function getRootElement(doc = document) {
        return doc.querySelector(ROOT_SELECTOR) || doc.body;
    }

    function executeScripts(container) {
        container.querySelectorAll('script').forEach(oldScript => {
            const newScript = document.createElement('script');
            Array.from(oldScript.attributes).forEach(attr => newScript.setAttribute(attr.name, attr.value));
            newScript.appendChild(document.createTextNode(oldScript.innerHTML));
            oldScript.parentNode.replaceChild(newScript, oldScript);
        });
    }

    async function navigateTo(url, pushState = true) {
        // Lifecycle: beforeNavigate
        const cancelEvent = new CustomEvent('pogonian:before-navigate', { cancelable: true, detail: { url } });
        if (!document.dispatchEvent(cancelEvent)) {
            return; // Navigation cancelled by listener
        }
        lifecycle.beforeNavigate.forEach(fn => { try { fn(url); } catch(e){} });

        try {
            document.body.classList.add('pogn-loading');
            const res = await fetch(url, {
                headers: {
                    'X-Pogonian-Request': 'true',
                    'Accept': 'text/html'
                }
            });

            if (!res.ok) {
                window.location.href = url;
                return;
            }

            const html = await res.text();
            const parser = new DOMParser();
            const newDoc = parser.parseFromString(html, 'text/html');

            // Lifecycle: beforeSwap
            document.dispatchEvent(new CustomEvent('pogonian:before-swap', { detail: { url, newDoc } }));
            lifecycle.beforeSwap.forEach(fn => { try { fn(url, newDoc); } catch(e){} });

            // 1. Reconcile Title
            if (newDoc.title) {
                document.title = newDoc.title;
            }

            // 2. Reconcile Head Meta
            newDoc.querySelectorAll('meta[name], meta[property]').forEach(newMeta => {
                const name = newMeta.getAttribute('name') || newMeta.getAttribute('property');
                const existing = document.querySelector(`meta[name="${name}"], meta[property="${name}"]`);
                if (existing) {
                    existing.setAttribute('content', newMeta.getAttribute('content'));
                }
            });

            // 3. Swap Root / Main content
            const currentRoot = getRootElement(document);
            const nextRoot = getRootElement(newDoc);

            if (currentRoot && nextRoot) {
                currentRoot.innerHTML = nextRoot.innerHTML;
                executeScripts(currentRoot);
            } else {
                document.body.innerHTML = newDoc.body.innerHTML;
                executeScripts(document.body);
            }

            // 4. Update active links in nav
            updateActiveLinks(url);

            // 5. Update History
            if (pushState) {
                window.history.pushState({ url }, '', url);
            }

            // 6. Scroll restoration
            window.scrollTo({ top: 0, behavior: 'instant' });

            // Lifecycle: afterNavigate
            document.dispatchEvent(new CustomEvent('pogonian:after-navigate', { detail: { url } }));
            document.dispatchEvent(new CustomEvent('pogonian:navigated', { detail: { url } }));
            lifecycle.afterNavigate.forEach(fn => { try { fn(url); } catch(e){} });

        } catch (err) {
            console.error('[Pogonian] Soft navigation failed, falling back to full load:', err);
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

        const href = link.getAttribute('href');
        const target = link.getAttribute('target');
        const download = link.hasAttribute('download');
        const rel = link.getAttribute('rel');

        if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
        if (target && target !== '_self') return;
        if (download || (rel && rel.includes('external'))) return;
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

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
        let method = (form.getAttribute('method') || 'GET').toUpperCase();
        const targetUrl = new URL(action, window.location.origin);

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
                const hasFileInput = form.querySelector('input[type="file"]') !== null;
                const isMultipart = (form.getAttribute('enctype') || '').includes('multipart/form-data') || hasFileInput;
                const formData = new FormData(form);

                let bodyPayload;
                let headers = {
                    'X-Pogonian-Request': 'true',
                    'Accept': 'text/html, application/json'
                };

                if (isMultipart) {
                    bodyPayload = formData; // Browser automatically adds boundary header
                } else {
                    bodyPayload = new URLSearchParams(formData);
                }

                res = await fetch(action, {
                    method: method,
                    body: bodyPayload,
                    headers: headers
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
                    executeScripts(currentRoot);
                }
                if (newDoc.title) document.title = newDoc.title;
            } else if (contentType.includes('application/json')) {
                const json = await res.json();
                document.dispatchEvent(new CustomEvent('pogonian:form-response', { detail: { data: json, status: res.status } }));
            }
        } catch (err) {
            console.error('[Pogonian] Form submission error:', err);
            form.submit();
        } finally {
            document.body.classList.remove('pogn-loading');
        }
    });

    window.addEventListener('popstate', () => {
        navigateTo(window.location.href, false);
    });

    document.addEventListener('DOMContentLoaded', () => {
        updateActiveLinks(window.location.href);
    });
})();
