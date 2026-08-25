/**
 * Pogonian Runtime (v0.3.0)
 * Architecture-aware SPA soft navigation, DOM patching, complete head reconciliation,
 * script lifecycle execution, method spoofing (PUT/PATCH/DELETE), multipart form handling,
 * and fail-closed CSRF token synchronization.
 */
(() => {
    'use strict';

    if (window.__pogonian_runtime_loaded) return;
    window.__pogonian_runtime_loaded = true;

    console.log("%c[Pogonian]%c Production-grade soft navigation runtime active", "color: #38bdf8; font-weight: bold;", "color: inherit;");

    const ROOT_SELECTOR = '#pogn-root, main, [data-pogn-root], #app';

    const lifecycle = {
        beforeNavigate: [],
        beforeSwap: [],
        afterNavigate: []
    };

    const api = {
        async request(url, options = {}) {
            const method = (options.method || 'GET').toUpperCase();
            const headers = {
                'X-Pogonian-Request': 'true',
                'Accept': 'application/json, text/plain, */*',
                ...(options.headers || {})
            };

            const csrf = getCSRFToken();
            if (csrf && !headers['X-CSRF-Token']) {
                headers['X-CSRF-Token'] = csrf;
            }

            let body = options.body;
            if (body && typeof body === 'object' && !(body instanceof FormData) && !(body instanceof URLSearchParams)) {
                headers['Content-Type'] = 'application/json';
                body = JSON.stringify(body);
            }

            const res = await fetch(url, {
                method,
                headers,
                body
            });

            const contentType = res.headers.get('content-type') || '';
            let data = null;
            if (contentType.includes('application/json')) {
                data = await res.json();
            } else {
                data = await res.text();
            }

            if (!res.ok) {
                const error = new Error(data && data.error ? data.error : `HTTP ${res.status} ${res.statusText}`);
                error.status = res.status;
                error.data = data;
                throw error;
            }

            return data;
        },

        get(url, options = {}) {
            return this.request(url, { ...options, method: 'GET' });
        },

        post(url, body, options = {}) {
            return this.request(url, { ...options, method: 'POST', body });
        },

        put(url, body, options = {}) {
            return this.request(url, { ...options, method: 'PUT', body });
        },

        delete(url, options = {}) {
            return this.request(url, { ...options, method: 'DELETE' });
        },

        entity(name) {
            const base = `/api/${name.toLowerCase()}`;
            return {
                list: (opts) => api.get(base, opts),
                get: (id, opts) => api.get(`${base}/${id}`, opts),
                create: (data, opts) => api.post(base, data, opts),
                update: (id, data, opts) => api.put(`${base}/${id}`, data, opts),
                remove: (id, opts) => api.delete(`${base}/${id}`, opts)
            };
        }
    };

    window.pogn = window.Pogonian = {
        version: '0.3.0',
        onBeforeNavigate: (fn) => lifecycle.beforeNavigate.push(fn),
        onBeforeSwap: (fn) => lifecycle.beforeSwap.push(fn),
        onAfterNavigate: (fn) => lifecycle.afterNavigate.push(fn),
        navigate: (url) => navigateTo(url, true),
        getCSRFToken: () => getCSRFToken(),
        api: api
    };

    function getCSRFToken() {
        // 1. From cookie
        const match = document.cookie.match(new RegExp('(^|;\\s*)_pogn_csrf=([^;]+)'));
        if (match) return decodeURIComponent(match[2]);

        // 2. From hidden form field
        const input = document.querySelector('input[name="_csrf"]');
        if (input && input.value) return input.value;

        // 3. From meta tag if present
        const meta = document.querySelector('meta[name="csrf-token"]');
        if (meta && meta.content) return meta.content;

        return '';
    }

    function getRootElement(doc = document) {
        return doc.querySelector(ROOT_SELECTOR) || doc.body;
    }

    function executeScripts(container) {
        const scripts = Array.from(container.querySelectorAll('script'));
        scripts.forEach(oldScript => {
            const newScript = document.createElement('script');
            Array.from(oldScript.attributes).forEach(attr => newScript.setAttribute(attr.name, attr.value));
            if (oldScript.src) {
                // External script
                newScript.src = oldScript.src;
            } else {
                // Inline script
                newScript.textContent = oldScript.textContent;
            }
            oldScript.parentNode.replaceChild(newScript, oldScript);
        });
    }

    function reconcileHead(newDoc) {
        // 1. Title
        if (newDoc.title) {
            document.title = newDoc.title;
        }

        // 2. Meta tags
        newDoc.querySelectorAll('meta').forEach(newMeta => {
            const name = newMeta.getAttribute('name');
            const property = newMeta.getAttribute('property');
            const httpEquiv = newMeta.getAttribute('http-equiv');

            let selector = '';
            if (name) selector = `meta[name="${name}"]`;
            else if (property) selector = `meta[property="${property}"]`;
            else if (httpEquiv) selector = `meta[http-equiv="${httpEquiv}"]`;

            if (selector) {
                const existing = document.head.querySelector(selector);
                if (existing) {
                    existing.setAttribute('content', newMeta.getAttribute('content') || '');
                } else {
                    document.head.appendChild(newMeta.cloneNode(true));
                }
            }
        });

        // 3. Stylesheets
        newDoc.querySelectorAll('link[rel="stylesheet"]').forEach(newLink => {
            const href = newLink.getAttribute('href');
            if (href && !document.head.querySelector(`link[rel="stylesheet"][href="${href}"]`)) {
                document.head.appendChild(newLink.cloneNode(true));
            }
        });
    }

    async function navigateTo(url, pushState = true) {
        // Lifecycle: beforeNavigate (cancellable)
        const cancelEvent = new CustomEvent('pogonian:before-navigate', { cancelable: true, detail: { url } });
        if (!document.dispatchEvent(cancelEvent)) {
            return; // Navigation cancelled
        }
        for (const fn of lifecycle.beforeNavigate) {
            try { if (fn(url) === false) return; } catch (e) { console.error(e); }
        }

        try {
            document.body.classList.add('pogn-loading');
            const csrfToken = getCSRFToken();

            const res = await fetch(url, {
                headers: {
                    'X-Pogonian-Request': 'true',
                    'X-CSRF-Token': csrfToken,
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

            // 1. Reconcile Head
            reconcileHead(newDoc);

            // 2. Swap Root / Main content
            const currentRoot = getRootElement(document);
            const nextRoot = getRootElement(newDoc);

            if (currentRoot && nextRoot) {
                currentRoot.innerHTML = nextRoot.innerHTML;
                executeScripts(currentRoot);
            } else {
                document.body.innerHTML = newDoc.body.innerHTML;
                executeScripts(document.body);
            }

            // 3. Update active links in nav
            updateActiveLinks(url);

            // 4. Update History
            if (pushState) {
                window.history.pushState({ url, scrollY: 0 }, '', url);
            }

            // 5. Scroll restoration
            const parsedUrl = new URL(url, window.location.origin);
            if (parsedUrl.hash) {
                const target = document.querySelector(parsedUrl.hash);
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth' });
                }
            } else {
                window.scrollTo({ top: 0, behavior: 'instant' });
            }

            // Lifecycle: afterNavigate
            document.dispatchEvent(new CustomEvent('pogonian:after-navigate', { detail: { url } }));
            document.dispatchEvent(new CustomEvent('pogonian:navigated', { detail: { url } }));
            lifecycle.afterNavigate.forEach(fn => { try { fn(url); } catch(e){} });

        } catch (err) {
            console.error('[Pogonian] Soft navigation error, fallback to full load:', err);
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

        if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:') || href.startsWith('javascript:')) return;
        if (target && target !== '_self') return;
        if (download || (rel && rel.includes('external'))) return;
        if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

        const targetUrl = new URL(href, window.location.origin);
        if (targetUrl.origin !== window.location.origin) return;

        e.preventDefault();
        navigateTo(targetUrl.href, true);
    });

    // Intercept form submissions (with PUT/PATCH/DELETE method spoofing & multipart support)
    document.addEventListener('submit', async (e) => {
        const form = e.target.closest('form');
        if (!form || form.dataset.pognNative !== undefined) return;

        const action = form.getAttribute('action') || window.location.href;
        let method = (form.getAttribute('method') || 'GET').toUpperCase();
        const spoofedMethod = form.querySelector('input[name="_method"]')?.value?.toUpperCase() || form.dataset.method?.toUpperCase();
        if (spoofedMethod) {
            method = spoofedMethod;
        }

        const targetUrl = new URL(action, window.location.origin);
        if (targetUrl.origin !== window.location.origin) return;

        e.preventDefault();
        document.body.classList.add('pogn-loading');

        try {
            if (method === 'GET') {
                const formData = new FormData(form);
                const params = new URLSearchParams(formData);
                targetUrl.search = params.toString();
                await navigateTo(targetUrl.href, true);
                return;
            }

            const hasFileInput = form.querySelector('input[type="file"]') !== null;
            const isMultipart = (form.getAttribute('enctype') || '').includes('multipart/form-data') || hasFileInput;
            const formData = new FormData(form);
            const csrfToken = getCSRFToken();

            let bodyPayload;
            let headers = {
                'X-Pogonian-Request': 'true',
                'X-CSRF-Token': csrfToken,
                'X-HTTP-Method-Override': method,
                'Accept': 'text/html, application/json'
            };

            if (isMultipart) {
                // If method spoofing with multipart, set _method in formData
                if (method !== 'POST') {
                    formData.set('_method', method);
                }
                bodyPayload = formData;
            } else {
                if (method !== 'POST') {
                    formData.set('_method', method);
                }
                bodyPayload = new URLSearchParams(formData);
            }

            const res = await fetch(action, {
                method: (method === 'GET' || method === 'POST') ? method : 'POST', // Fetch with POST and spoof header/body
                body: bodyPayload,
                headers: headers
            });

            if (res.redirected) {
                await navigateTo(res.url, true);
                return;
            }

            const contentType = res.headers.get('content-type') || '';
            if (contentType.includes('text/html')) {
                const html = await res.text();
                const parser = new DOMParser();
                const newDoc = parser.parseFromString(html, 'text/html');
                
                reconcileHead(newDoc);
                const currentRoot = getRootElement(document);
                const nextRoot = getRootElement(newDoc);
                if (currentRoot && nextRoot) {
                    currentRoot.innerHTML = nextRoot.innerHTML;
                    executeScripts(currentRoot);
                }
            } else if (contentType.includes('application/json')) {
                const json = await res.json();
                document.dispatchEvent(new CustomEvent('pogonian:form-response', { detail: { data: json, status: res.status } }));
            }
        } catch (err) {
            console.error('[Pogonian] Form submission error, falling back:', err);
            form.submit();
        } finally {
            document.body.classList.remove('pogn-loading');
        }
    });

    window.addEventListener('popstate', (e) => {
        navigateTo(window.location.href, false);
    });

    document.addEventListener('DOMContentLoaded', () => {
        updateActiveLinks(window.location.href);
    });
})();
